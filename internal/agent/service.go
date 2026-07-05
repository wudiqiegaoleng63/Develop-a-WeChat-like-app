package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"kama-chat-server/internal/dao"
	"kama-chat-server/internal/dto/request"
	"kama-chat-server/internal/model"
	"kama-chat-server/pkg/constants"
	"kama-chat-server/pkg/enum/message/message_status_enum"
	"kama-chat-server/pkg/enum/message/message_type_enum"
	"kama-chat-server/pkg/util/random"
	"kama-chat-server/pkg/zlog"

	"github.com/go-redis/redis/v8"
	myredis "kama-chat-server/internal/service/redis"
)

// AgentService 封装 AI 助手在 IM 系统中的业务逻辑。
// 不破坏现有单聊/群聊路由，仅在外部以协程方式触发。
var AgentService = newAgentService()

type agentService struct{}

func newAgentService() *agentService { return &agentService{} }

// ChatWithPrivateAgent 私聊 Agent：用户给 Agent 发消息后调用。
// userID 为提问用户 uuid，content 为用户输入。
// 该方法会：构建上下文 → 调用 LLM → 持久化 Agent 回复消息。
// 推送由调用方（chat server）通过返回的 messageBack 完成，避免反向依赖。
func (a *agentService) ChatWithPrivateAgent(ctx context.Context, userID, content string) (string, error) {
	start := time.Now()

	// 1. 输入长度限制，防超长 prompt
	if len(content) > constants.AgentMaxInputLen {
		content = content[:constants.AgentMaxInputLen]
	}

	// 2. 构建上下文（system + 最近 N 轮历史 + 当前问题）
	msgs := a.buildPrivateContext(userID, content)

	// 3. 调用 LLM（带超时）
	reply, err := DefaultClient.Generate(ctx, msgs)
	cost := time.Since(start).Milliseconds()
	if err != nil {
		zlog.Error(fmt.Sprintf("Agent private: userID=%s cost=%dms err=%s", userID, cost, err.Error()))
		return constants.AgentErrMsg, err
	}
	zlog.Info(fmt.Sprintf("Agent private: userID=%s cost=%dms replyLen=%d", userID, cost, len(reply)))
	return reply, nil
}

// ChatWithGroupAgent 群聊 Agent：群消息命中触发词后调用。
// groupID 群 uuid，userID 提问人 uuid，content 已剥离触发词的正文。
func (a *agentService) ChatWithGroupAgent(ctx context.Context, groupID, userID, content string) (string, error) {
	start := time.Now()

	if len(content) > constants.AgentMaxInputLen {
		content = content[:constants.AgentMaxInputLen]
	}

	msgs := a.buildGroupContext(groupID, userID, content)

	reply, err := DefaultClient.Generate(ctx, msgs)
	cost := time.Since(start).Milliseconds()
	if err != nil {
		zlog.Error(fmt.Sprintf("Agent group: groupID=%s userID=%s cost=%dms err=%s", groupID, userID, cost, err.Error()))
		return constants.AgentErrMsg, err
	}

	// 群聊回复前缀 @提问人，避免上下文混乱
	nickname := a.lookupNickname(userID)
	if nickname != "" {
		reply = "@" + nickname + " " + reply
	}
	zlog.Info(fmt.Sprintf("Agent group: groupID=%s userID=%s cost=%dms replyLen=%d", groupID, userID, cost, len(reply)))
	return reply, nil
}

// buildPrivateContext 构建私聊上下文：system + 最近 N 条历史 + 当前问题。
// 只读取当前用户与 Agent 的历史，不涉及他人私聊。
func (a *agentService) buildPrivateContext(userID, content string) []Message {
	msgs := []Message{
		{Role: RoleSystem, Content: "你是 KamaChat 的 AI 助手，用简洁的中文回答用户问题。"},
	}

	history := a.lastNPrivateMessages(userID, constants.AgentBotUuid, constants.AgentPrivateContextLen)
	for _, m := range history {
		msgs = append(msgs, m)
	}

	msgs = append(msgs, Message{Role: RoleUser, Content: content})
	return msgs
}

// buildGroupContext 构建群聊上下文：system + 最近 N 条群历史 + 当前问题。
// 只读取当前群消息，不读取其他群。
func (a *agentService) buildGroupContext(groupID, userID, content string) []Message {
	msgs := []Message{
		{Role: RoleSystem, Content: "你是群里的 AI 助手，根据群聊上下文简短回答被 @ 的问题。"},
	}

	history := a.lastNGroupMessages(groupID, constants.AgentGroupContextLen)
	for _, m := range history {
		msgs = append(msgs, m)
	}

	msgs = append(msgs, Message{Role: RoleUser, Content: content})
	return msgs
}

// lastNPrivateMessages 读取用户与 Agent 最近 N 条消息，转为 LLM Message。
// 发送方为用户 → user 角色；发送方为 Agent → assistant 角色。
func (a *agentService) lastNPrivateMessages(userID, agentID string, n int) []Message {
	var list []model.Message
	// 取最近 n 条（按时间倒序），再在内存中翻转为正序
	if res := dao.GormDB.Where(
		"(send_id = ? AND receive_id = ?) OR (send_id = ? AND receive_id = ?)",
		userID, agentID, agentID, userID,
	).Order("created_at DESC").Limit(n).Find(&list); res.Error != nil {
		zlog.Error("Agent lastNPrivate: " + res.Error.Error())
		return nil
	}
	// 翻转为正序
	for i, j := 0, len(list)-1; i < j; i, j = i+1, j-1 {
		list[i], list[j] = list[j], list[i]
	}
	msgs := make([]Message, 0, len(list))
	for _, m := range list {
		if m.SendId == agentID {
			msgs = append(msgs, Message{Role: RoleAssistant, Content: m.Content})
		} else {
			msgs = append(msgs, Message{Role: RoleUser, Content: m.Content})
		}
	}
	return msgs
}

// lastNGroupMessages 读取群最近 N 条消息，转为 LLM Message。
// Agent 自己发的视为 assistant，群成员发的视为 user（带昵称前缀）。
func (a *agentService) lastNGroupMessages(groupID string, n int) []Message {
	var list []model.Message
	if res := dao.GormDB.Where("receive_id = ?", groupID).
		Order("created_at DESC").Limit(n).Find(&list); res.Error != nil {
		zlog.Error("Agent lastNGroup: " + res.Error.Error())
		return nil
	}
	for i, j := 0, len(list)-1; i < j; i, j = i+1, j-1 {
		list[i], list[j] = list[j], list[i]
	}
	msgs := make([]Message, 0, len(list))
	for _, m := range list {
		if m.SendId == constants.AgentBotUuid {
			msgs = append(msgs, Message{Role: RoleAssistant, Content: m.Content})
		} else {
			msgs = append(msgs, Message{Role: RoleUser, Content: m.SendName + ": " + m.Content})
		}
	}
	return msgs
}

// lookupNickname 查询用户昵称，群聊回复 @提问人 时使用。
func (a *agentService) lookupNickname(userID string) string {
	var u model.UserInfo
	if res := dao.GormDB.Where("uuid = ?", userID).First(&u); res.Error != nil {
		return ""
	}
	return u.Nickname
}

// MatchGroupTrigger 判断群消息是否触发 Agent，命中则返回剥离触发词后的正文。
// 未命中返回 ("", false)。
func MatchGroupTrigger(content string) (string, bool) {
	for _, t := range constants.AgentGroupTriggers {
		if strings.HasPrefix(content, t) {
			return strings.TrimSpace(strings.TrimPrefix(content, t)), true
		}
	}
	return "", false
}

// IsAgentTarget 判断私聊接收方是否为系统 Agent。
func IsAgentTarget(receiveID string) bool {
	return receiveID == constants.AgentBotUuid
}

// BuildAgentReplyMessage 构造一条 Agent 回复消息（持久化 + 推送复用）。
// 调用方负责把该消息推送给目标用户/群成员。
func BuildAgentReplyMessage(receiveID string, reply string) (model.Message, request.ChatMessageRequest) {
	uuid, _ := random.GetNowAndLenRandomString(11)
	msg := model.Message{
		Uuid:       "M" + uuid,
		Type:       message_type_enum.Text,
		Content:    reply,
		SendId:     constants.AgentBotUuid,
		SendName:   constants.AgentBotNickname,
		SendAvatar: constants.AgentBotAvatar,
		ReceiveId:  receiveID,
		FileSize:   "0B",
		Status:     message_status_enum.Sent, // Agent 回复直接置为已发送
		CreatedAt:  time.Now(),
	}
	req := request.ChatMessageRequest{
		Type:       message_type_enum.Text,
		Content:    reply,
		SendId:     constants.AgentBotUuid,
		SendName:   constants.AgentBotNickname,
		SendAvatar: constants.AgentBotAvatar,
		ReceiveId:  receiveID,
	}
	return msg, req
}

// PersistMessage 持久化 Agent 回复消息到 message 表，供历史消息接口查询。
func PersistMessage(msg *model.Message) {
	if res := dao.GormDB.Create(msg); res.Error != nil {
		zlog.Error("Agent persist: " + res.Error.Error())
	}
}

// marshalReply 序列化推送载荷，供 chat server 复用其 SendMessageToClient 逻辑。
// 此处仅做工具方法，实际推送由 chat 层完成，避免 agent 反向依赖 chat 包。
func marshalReply(v interface{}) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		zlog.Error("Agent marshal: " + err.Error())
		return nil
	}
	return b
}

var _ = marshalReply // 预留：流式/自定义推送时使用

// PrivateReply 私聊 Agent 回复产物：已持久化的消息 + 推送给用户的载荷。
type PrivateReply struct {
	Message model.Message
	Payload []byte // respond.GetMessageListRespond 序列化结果
}

// GroupReply 群聊 Agent 回复产物：已持久化的消息 + 推送给群成员的载荷。
type GroupReply struct {
	Message model.Message
	Payload []byte // respond.GetGroupMessageListRespond 序列化结果
}

// TriggerPrivate 私聊触发 Agent。chat 层在私聊消息持久化并推送后，用协程调用本方法，
// 再把返回的 Payload 推送给用户（参考现有私聊推送逻辑）。
// 内部完成：构建上下文 → 调用 LLM → 持久化 Agent 回复 → 序列化推送载荷。
func (a *agentService) TriggerPrivate(ctx context.Context, userID string) *PrivateReply {
	content := a.lastUserContentToAgent(userID)
	if content == "" {
		return nil
	}
	reply, err := a.ChatWithPrivateAgent(ctx, userID, content)
	if err != nil {
		reply = constants.AgentErrMsg
	}

	msg, _ := BuildAgentReplyMessage(userID, reply)
	PersistMessage(&msg)

	// 推送载荷与现有 GetMessageListRespond 结构一致
	payload := marshalReply(struct {
		SendId     string `json:"send_id"`
		SendName   string `json:"send_name"`
		SendAvatar string `json:"send_avatar"`
		ReceiveId  string `json:"receive_id"`
		Type       int8   `json:"type"`
		Content    string `json:"content"`
		Url        string `json:"url"`
		FileSize   string `json:"file_size"`
		FileName   string `json:"file_name"`
		FileType   string `json:"file_type"`
		CreatedAt  string `json:"created_at"`
	}{
		SendId:     msg.SendId,
		SendName:   msg.SendName,
		SendAvatar: msg.SendAvatar,
		ReceiveId:  msg.ReceiveId,
		Type:       msg.Type,
		Content:    msg.Content,
		FileSize:   msg.FileSize,
		CreatedAt:  msg.CreatedAt.Format("2006-01-02 15:04:05"),
	})

	// 更新私聊 redis 缓存（与现有 message_list_<send>_<receive> 一致）
	a.appendPrivateCache(userID, constants.AgentBotUuid, payload)

	return &PrivateReply{Message: msg, Payload: payload}
}

// TriggerGroup 群聊触发 Agent。chat 层在群消息持久化并推送后，用协程调用本方法，
// 再把返回的 Payload 群发给除 Agent 外的在线成员。
func (a *agentService) TriggerGroup(ctx context.Context, groupID, userID, rawContent string) *GroupReply {
	content, ok := MatchGroupTrigger(rawContent)
	if !ok {
		return nil
	}
	reply, err := a.ChatWithGroupAgent(ctx, groupID, userID, content)
	if err != nil {
		reply = constants.AgentErrMsg
	}

	msg, _ := BuildAgentReplyMessage(groupID, reply)
	msg.SendId = constants.AgentBotUuid
	PersistMessage(&msg)

	payload := marshalReply(struct {
		SendId     string `json:"send_id"`
		SendName   string `json:"send_name"`
		SendAvatar string `json:"send_avatar"`
		ReceiveId  string `json:"receive_id"`
		Type       int8   `json:"type"`
		Content    string `json:"content"`
		Url        string `json:"url"`
		FileSize   string `json:"file_size"`
		FileName   string `json:"file_name"`
		FileType   string `json:"file_type"`
		CreatedAt  string `json:"created_at"`
	}{
		SendId:     msg.SendId,
		SendName:   msg.SendName,
		SendAvatar: msg.SendAvatar,
		ReceiveId:  msg.ReceiveId,
		Type:       msg.Type,
		Content:    msg.Content,
		FileSize:   msg.FileSize,
		CreatedAt:  msg.CreatedAt.Format("2006-01-02 15:04:05"),
	})

	a.appendGroupCache(groupID, payload)
	return &GroupReply{Message: msg, Payload: payload}
}

// lastUserContentToAgent 取用户最近发给 Agent 的一条消息正文，作为本次提问。
// 由 chat 层在持久化用户消息后触发，因此该消息一定已落库。
func (a *agentService) lastUserContentToAgent(userID string) string {
	var m model.Message
	if res := dao.GormDB.Where("send_id = ? AND receive_id = ?", userID, constants.AgentBotUuid).
		Order("created_at DESC").First(&m); res.Error != nil {
		return ""
	}
	return m.Content
}

// appendPrivateCache 追加 Agent 回复到私聊 redis 缓存。
// 与 message_service.go 中 message_list_<send>_<receive> 的方向保持一致。
func (a *agentService) appendPrivateCache(userID, agentID string, payload []byte) {
	key := "message_list_" + agentID + "_" + userID
	a.appendCacheList(key, payload)
}

// appendGroupCache 追加 Agent 回复到群聊 redis 缓存 group_messagelist_<group>。
func (a *agentService) appendGroupCache(groupID string, payload []byte) {
	key := "group_messagelist_" + groupID
	a.appendCacheList(key, payload)
}

// appendCacheList 读取 redis 列表缓存，追加一条载荷后回写。
// 与现有 message_service.go 的缓存读写模式保持一致（1 分钟 TTL）。
// 缓存不存在时静默跳过——下次 GetMessageList 查询会重建缓存。
func (a *agentService) appendCacheList(key string, payload []byte) {
	if len(payload) == 0 {
		return
	}
	raw, err := myredis.GetKeyNilIsErr(key)
	if err != nil {
		if !errors.Is(err, redis.Nil) {
			zlog.Error("Agent appendCache: " + err.Error())
		}
		return
	}
	// 现有缓存是一个 JSON 数组，直接拼接以避免反序列化/序列化开销
	trimmed := strings.TrimRight(raw, "]")
	if trimmed == raw {
		// 不是合法数组，放弃追加，等下次查询重建
		return
	}
	var newRaw string
	if trimmed == strings.TrimRight(raw, " ]") {
		// 空数组 "[]"
		newRaw = "[" + string(payload) + "]"
	} else {
		newRaw = trimmed + "," + string(payload) + "]"
	}
	if err := myredis.SetKeyEx(key, newRaw, time.Minute*constants.REDIS_TIMEOUT); err != nil {
		zlog.Error("Agent appendCache set: " + err.Error())
	}
}
