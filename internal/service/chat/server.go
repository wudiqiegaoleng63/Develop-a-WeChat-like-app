package chat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/gorilla/websocket"
	"kama-chat-server/internal/agent"
	"kama-chat-server/internal/dao"
	"kama-chat-server/internal/dto/request"
	"kama-chat-server/internal/dto/respond"
	"kama-chat-server/internal/model"
	myredis "kama-chat-server/internal/service/redis"
	"kama-chat-server/pkg/constants"
	"kama-chat-server/pkg/enum/message/message_status_enum"
	"kama-chat-server/pkg/enum/message/message_type_enum"
	"kama-chat-server/pkg/util/random"
	"kama-chat-server/pkg/zlog"
	"log"
	"strings"
	"sync"
	"time"
)

type Server struct {
	Clients  map[string]*Client
	mutex    *sync.Mutex
	Transmit chan []byte  // 转发通道
	Login    chan *Client // 登录通道
	Logout   chan *Client // 退出登录通道
}

var ChatServer *Server

func init() {
	if ChatServer == nil {
		ChatServer = &Server{
			Clients:  make(map[string]*Client),
			mutex:    &sync.Mutex{},
			Transmit: make(chan []byte, constants.CHANNEL_SIZE),
			Login:    make(chan *Client, constants.CHANNEL_SIZE),
			Logout:   make(chan *Client, constants.CHANNEL_SIZE),
		}
	}
}

// 将https://127.0.0.1:8000/static/xxx 转为 /static/xxx
func normalizePath(path string) string {
	// 查找 "/static/" 的位置
	if path == "https://cube.elemecdn.com/0/88/03b0d39583f48206768a7534e55bcpng.png" {
		return path
	}
	staticIndex := strings.Index(path, "/static/")
	if staticIndex < 0 {
		log.Println(path)
		zlog.Error("路径不合法")
	}
	// 返回从 "/static/" 开始的部分
	return path[staticIndex:]
}

// Start 启动函数，Server端用主进程起，Client端可以用协程起
func (s *Server) Start() {
	defer func() {
		close(s.Transmit)
		close(s.Logout)
		close(s.Login)
	}()
	for {
		select {
		case client := <-s.Login:
			{
				s.mutex.Lock()
				s.Clients[client.Uuid] = client
				s.mutex.Unlock()
				zlog.Debug(fmt.Sprintf("欢迎来到GoChat聊天服务器，亲爱的用户%s\n", client.Uuid))
				err := client.Conn.WriteMessage(websocket.TextMessage, []byte("欢迎来到GoChat聊天服务器"))
				if err != nil {
					zlog.Error(err.Error())
				}
			}

		case client := <-s.Logout:
			{
				s.mutex.Lock()
				delete(s.Clients, client.Uuid)
				s.mutex.Unlock()
				zlog.Info(fmt.Sprintf("用户%s退出登录\n", client.Uuid))
				if err := client.Conn.WriteMessage(websocket.TextMessage, []byte("已退出登录")); err != nil {
					zlog.Error(err.Error())
				}
			}

		case data := <-s.Transmit:
			{
				var chatMessageReq request.ChatMessageRequest
				if err := json.Unmarshal(data, &chatMessageReq); err != nil {
					zlog.Error(err.Error())
				}
				// log.Println("原消息为：", data, "反序列化后为：", chatMessageReq)
				if chatMessageReq.Type == message_type_enum.Text {
					// 存message
					value, err := random.GetNowAndLenRandomString(11)
					if err != nil {
						zlog.Error(err.Error())
					}
					message := model.Message{
						Uuid:       fmt.Sprintf("M%s", value),
						SessionId:  chatMessageReq.SessionId,
						Type:       chatMessageReq.Type,
						Content:    chatMessageReq.Content,
						Url:        "",
						SendId:     chatMessageReq.SendId,
						SendName:   chatMessageReq.SendName,
						SendAvatar: chatMessageReq.SendAvatar,
						ReceiveId:  chatMessageReq.ReceiveId,
						FileSize:   "0B",
						FileType:   "",
						FileName:   "",
						Status:     message_status_enum.Unsent,
						CreatedAt:  time.Now(),
						AVdata:     "",
					}
					// 对SendAvatar去除前面/static之前的所有内容，防止ip前缀引入
					message.SendAvatar = normalizePath(message.SendAvatar)
					if res := dao.GormDB.Create(&message); res.Error != nil {
						zlog.Error(res.Error.Error())
					}
					if message.ReceiveId[0] == 'U' { // 发送给User
						// 如果能找到ReceiveId，说明在线，可以发送，否则存表后跳过
						// 因为在线的时候是通过websocket更新消息记录的，离线后通过存表，登录时只调用一次数据库操作
						// 切换chat对象后，前端的messageList也会改变，获取messageList从第二次就是从redis中获取
						messageRsp := respond.GetMessageListRespond{
							SendId:     message.SendId,
							SendName:   message.SendName,
							SendAvatar: chatMessageReq.SendAvatar,
							ReceiveId:  message.ReceiveId,
							Type:       message.Type,
							Content:    message.Content,
							Url:        message.Url,
							FileSize:   message.FileSize,
							FileName:   message.FileName,
							FileType:   message.FileType,
							CreatedAt:  message.CreatedAt.Format("2006-01-02 15:04:05"),
						}
						jsonMessage, err := json.Marshal(messageRsp)
						if err != nil {
							zlog.Error(err.Error())
						}
						log.Println("返回的消息为：", messageRsp, "序列化后为：", jsonMessage)
						var messageBack = &MessageBack{
							Message: jsonMessage,
							Uuid:    message.Uuid,
						}
						s.mutex.Lock()
						if receiveClient, ok := s.Clients[message.ReceiveId]; ok {
							//messageBack.Message = jsonMessage
							//messageBack.Uuid = message.Uuid
							receiveClient.SendBack <- messageBack // 向client.Send发送
						}
						// 因为send_id肯定在线，所以这里在后端进行在线回显message，其实优化的话前端可以直接回显
						// 问题在于前后端的req和rsp结构不同，前端存储message的messageList不能存req，只能存rsp
						// 所以这里后端进行回显，前端不回显
						sendClient := s.Clients[message.SendId]
						sendClient.SendBack <- messageBack
						s.mutex.Unlock()

						// redis
						var rspString string
						rspString, err = myredis.GetKeyNilIsErr("message_list_" + message.SendId + "_" + message.ReceiveId)
						if err == nil {
							var rsp []respond.GetMessageListRespond
							if err := json.Unmarshal([]byte(rspString), &rsp); err != nil {
								zlog.Error(err.Error())
							}
							rsp = append(rsp, messageRsp)
							rspByte, err := json.Marshal(rsp)
							if err != nil {
								zlog.Error(err.Error())
							}
							if err := myredis.SetKeyEx("message_list_"+message.SendId+"_"+message.ReceiveId, string(rspByte), time.Minute*constants.REDIS_TIMEOUT); err != nil {
								zlog.Error(err.Error())
							}
						} else {
							if !errors.Is(err, redis.Nil) {
								zlog.Error(err.Error())
							}
						}

						// ===== Agent 私聊触发 =====
						// 接收方为系统 Agent 时，协程触发 LLM 回复，回复复用现有推送通道
						if agent.IsAgentTarget(message.ReceiveId) {
							go s.triggerPrivateAgent(message.SendId)
						}
					} else if message.ReceiveId[0] == 'G' { // 发送给Group
						messageRsp := respond.GetGroupMessageListRespond{
							SendId:     message.SendId,
							SendName:   message.SendName,
							SendAvatar: chatMessageReq.SendAvatar,
							ReceiveId:  message.ReceiveId,
							Type:       message.Type,
							Content:    message.Content,
							Url:        message.Url,
							FileSize:   message.FileSize,
							FileName:   message.FileName,
							FileType:   message.FileType,
							CreatedAt:  message.CreatedAt.Format("2006-01-02 15:04:05"),
						}
						jsonMessage, err := json.Marshal(messageRsp)
						if err != nil {
							zlog.Error(err.Error())
						}
						log.Println("返回的消息为：", messageRsp, "序列化后为：", jsonMessage)
						var messageBack = &MessageBack{
							Message: jsonMessage,
							Uuid:    message.Uuid,
						}
						var group model.GroupInfo
						if res := dao.GormDB.Where("uuid = ?", message.ReceiveId).First(&group); res.Error != nil {
							zlog.Error(res.Error.Error())
						}
						var members []string
						if err := json.Unmarshal(group.Members, &members); err != nil {
							zlog.Error(err.Error())
						}
						s.mutex.Lock()
						for _, member := range members {
							if member != message.SendId {
								if receiveClient, ok := s.Clients[member]; ok {
									receiveClient.SendBack <- messageBack
								}
							} else {
								sendClient := s.Clients[message.SendId]
								sendClient.SendBack <- messageBack
							}
						}
						s.mutex.Unlock()

						// redis
						var rspString string
						rspString, err = myredis.GetKeyNilIsErr("group_messagelist_" + message.ReceiveId)
						if err == nil {
							var rsp []respond.GetGroupMessageListRespond
							if err := json.Unmarshal([]byte(rspString), &rsp); err != nil {
								zlog.Error(err.Error())
							}
							rsp = append(rsp, messageRsp)
							rspByte, err := json.Marshal(rsp)
							if err != nil {
								zlog.Error(err.Error())
							}
							if err := myredis.SetKeyEx("group_messagelist_"+message.ReceiveId, string(rspByte), time.Minute*constants.REDIS_TIMEOUT); err != nil {
								zlog.Error(err.Error())
							}
						} else {
							if !errors.Is(err, redis.Nil) {
								zlog.Error(err.Error())
							}
						}
					}
					// ===== Agent 群聊触发 =====
					// 消息命中 @AI助手 / /ai 等触发词时，协程触发 LLM 回复
					if _, ok := agent.MatchGroupTrigger(message.Content); ok {
						go s.triggerGroupAgent(message.ReceiveId, message.SendId, message.Content)
					}
				} else if chatMessageReq.Type == message_type_enum.File {
					// 存message
					value, err := random.GetNowAndLenRandomString(11)
					if err != nil {
						zlog.Error(err.Error())
					}
					message := model.Message{
						Uuid:       fmt.Sprintf("M%s", value),
						SessionId:  chatMessageReq.SessionId,
						Type:       chatMessageReq.Type,
						Content:    "",
						Url:        chatMessageReq.Url,
						SendId:     chatMessageReq.SendId,
						SendName:   chatMessageReq.SendName,
						SendAvatar: chatMessageReq.SendAvatar,
						ReceiveId:  chatMessageReq.ReceiveId,
						FileSize:   chatMessageReq.FileSize,
						FileType:   chatMessageReq.FileType,
						FileName:   chatMessageReq.FileName,
						Status:     message_status_enum.Unsent,
						CreatedAt:  time.Now(),
						AVdata:     "",
					}
					// 对SendAvatar去除前面/static之前的所有内容，防止ip前缀引入
					message.SendAvatar = normalizePath(message.SendAvatar)
					if res := dao.GormDB.Create(&message); res.Error != nil {
						zlog.Error(res.Error.Error())
					}
					if message.ReceiveId[0] == 'U' { // 发送给User
						// 如果能找到ReceiveId，说明在线，可以发送，否则存表后跳过
						// 因为在线的时候是通过websocket更新消息记录的，离线后通过存表，登录时只调用一次数据库操作
						// 切换chat对象后，前端的messageList也会改变，获取messageList从第二次就是从redis中获取
						messageRsp := respond.GetMessageListRespond{
							SendId:     message.SendId,
							SendName:   message.SendName,
							SendAvatar: chatMessageReq.SendAvatar,
							ReceiveId:  message.ReceiveId,
							Type:       message.Type,
							Content:    message.Content,
							Url:        message.Url,
							FileSize:   message.FileSize,
							FileName:   message.FileName,
							FileType:   message.FileType,
							CreatedAt:  message.CreatedAt.Format("2006-01-02 15:04:05"),
						}
						jsonMessage, err := json.Marshal(messageRsp)
						if err != nil {
							zlog.Error(err.Error())
						}
						log.Println("返回的消息为：", messageRsp, "序列化后为：", jsonMessage)
						var messageBack = &MessageBack{
							Message: jsonMessage,
							Uuid:    message.Uuid,
						}
						s.mutex.Lock()
						if receiveClient, ok := s.Clients[message.ReceiveId]; ok {
							//messageBack.Message = jsonMessage
							//messageBack.Uuid = message.Uuid
							receiveClient.SendBack <- messageBack // 向client.Send发送
						}
						// 因为send_id肯定在线，所以这里在后端进行在线回显message，其实优化的话前端可以直接回显
						// 问题在于前后端的req和rsp结构不同，前端存储message的messageList不能存req，只能存rsp
						// 所以这里后端进行回显，前端不回显
						sendClient := s.Clients[message.SendId]
						sendClient.SendBack <- messageBack
						s.mutex.Unlock()

						// redis
						var rspString string
						rspString, err = myredis.GetKeyNilIsErr("message_list_" + message.SendId + "_" + message.ReceiveId)
						if err == nil {
							var rsp []respond.GetMessageListRespond
							if err := json.Unmarshal([]byte(rspString), &rsp); err != nil {
								zlog.Error(err.Error())
							}
							rsp = append(rsp, messageRsp)
							rspByte, err := json.Marshal(rsp)
							if err != nil {
								zlog.Error(err.Error())
							}
							if err := myredis.SetKeyEx("message_list_"+message.SendId+"_"+message.ReceiveId, string(rspByte), time.Minute*constants.REDIS_TIMEOUT); err != nil {
								zlog.Error(err.Error())
							}
						} else {
							if !errors.Is(err, redis.Nil) {
								zlog.Error(err.Error())
							}
						}
					} else {
						messageRsp := respond.GetGroupMessageListRespond{
							SendId:     message.SendId,
							SendName:   message.SendName,
							SendAvatar: chatMessageReq.SendAvatar,
							ReceiveId:  message.ReceiveId,
							Type:       message.Type,
							Content:    message.Content,
							Url:        message.Url,
							FileSize:   message.FileSize,
							FileName:   message.FileName,
							FileType:   message.FileType,
							CreatedAt:  message.CreatedAt.Format("2006-01-02 15:04:05"),
						}
						jsonMessage, err := json.Marshal(messageRsp)
						if err != nil {
							zlog.Error(err.Error())
						}
						log.Println("返回的消息为：", messageRsp, "序列化后为：", jsonMessage)
						var messageBack = &MessageBack{
							Message: jsonMessage,
							Uuid:    message.Uuid,
						}
						var group model.GroupInfo
						if res := dao.GormDB.Where("uuid = ?", message.ReceiveId).First(&group); res.Error != nil {
							zlog.Error(res.Error.Error())
						}
						var members []string
						if err := json.Unmarshal(group.Members, &members); err != nil {
							zlog.Error(err.Error())
						}
						s.mutex.Lock()
						for _, member := range members {
							if member != message.SendId {
								if receiveClient, ok := s.Clients[member]; ok {
									receiveClient.SendBack <- messageBack
								}
							} else {
								sendClient := s.Clients[message.SendId]
								sendClient.SendBack <- messageBack
							}
						}
						s.mutex.Unlock()

						// redis
						var rspString string
						rspString, err = myredis.GetKeyNilIsErr("group_messagelist_" + message.ReceiveId)
						if err == nil {
							var rsp []respond.GetGroupMessageListRespond
							if err := json.Unmarshal([]byte(rspString), &rsp); err != nil {
								zlog.Error(err.Error())
							}
							rsp = append(rsp, messageRsp)
							rspByte, err := json.Marshal(rsp)
							if err != nil {
								zlog.Error(err.Error())
							}
							if err := myredis.SetKeyEx("group_messagelist_"+message.ReceiveId, string(rspByte), time.Minute*constants.REDIS_TIMEOUT); err != nil {
								zlog.Error(err.Error())
							}
						} else {
							if !errors.Is(err, redis.Nil) {
								zlog.Error(err.Error())
							}
						}
					}
				} else if chatMessageReq.Type == message_type_enum.AudioOrVideo {
					var avData request.AVData
					if err := json.Unmarshal([]byte(chatMessageReq.AVdata), &avData); err != nil {
						zlog.Error(err.Error())
					}
					//log.Println(avData)
					value, err := random.GetNowAndLenRandomString(11)
					if err != nil {
						zlog.Error(err.Error())
					}
					message := model.Message{
						Uuid:       fmt.Sprintf("M%s", value),
						SessionId:  chatMessageReq.SessionId,
						Type:       chatMessageReq.Type,
						Content:    "",
						Url:        "",
						SendId:     chatMessageReq.SendId,
						SendName:   chatMessageReq.SendName,
						SendAvatar: chatMessageReq.SendAvatar,
						ReceiveId:  chatMessageReq.ReceiveId,
						FileSize:   "",
						FileType:   "",
						FileName:   "",
						Status:     message_status_enum.Unsent,
						CreatedAt:  time.Now(),
						AVdata:     chatMessageReq.AVdata,
					}
					if avData.MessageId == "PROXY" && (avData.Type == "start_call" || avData.Type == "receive_call" || avData.Type == "reject_call") {
						// 存message
						// 对SendAvatar去除前面/static之前的所有内容，防止ip前缀引入
						message.SendAvatar = normalizePath(message.SendAvatar)
						if res := dao.GormDB.Create(&message); res.Error != nil {
							zlog.Error(res.Error.Error())
						}
					}

					if chatMessageReq.ReceiveId[0] == 'U' { // 发送给User
						// 如果能找到ReceiveId，说明在线，可以发送，否则存表后跳过
						// 因为在线的时候是通过websocket更新消息记录的，离线后通过存表，登录时只调用一次数据库操作
						// 切换chat对象后，前端的messageList也会改变，获取messageList从第二次就是从redis中获取
						messageRsp := respond.AVMessageRespond{
							SendId:     message.SendId,
							SendName:   message.SendName,
							SendAvatar: message.SendAvatar,
							ReceiveId:  message.ReceiveId,
							Type:       message.Type,
							Content:    message.Content,
							Url:        message.Url,
							FileSize:   message.FileSize,
							FileName:   message.FileName,
							FileType:   message.FileType,
							CreatedAt:  message.CreatedAt.Format("2006-01-02 15:04:05"),
							AVdata:     message.AVdata,
						}
						jsonMessage, err := json.Marshal(messageRsp)
						if err != nil {
							zlog.Error(err.Error())
						}
						// log.Println("返回的消息为：", messageRsp, "序列化后为：", jsonMessage)
						log.Println("返回的消息为：", messageRsp)
						var messageBack = &MessageBack{
							Message: jsonMessage,
							Uuid:    message.Uuid,
						}
						s.mutex.Lock()
						if receiveClient, ok := s.Clients[message.ReceiveId]; ok {
							//messageBack.Message = jsonMessage
							//messageBack.Uuid = message.Uuid
							receiveClient.SendBack <- messageBack // 向client.Send发送
						}
						// 通话这不能回显，发回去的话就会出现两个start_call。
						//sendClient := s.Clients[message.SendId]
						//sendClient.SendBack <- messageBack
						s.mutex.Unlock()
					}
				}

			}
		}
	}
}

func (s *Server) Close() {
	close(s.Login)
	close(s.Logout)
	close(s.Transmit)
}

func (s *Server) SendClientToLogin(client *Client) {
	s.mutex.Lock()
	s.Login <- client
	s.mutex.Unlock()
}

func (s *Server) SendClientToLogout(client *Client) {
	s.mutex.Lock()
	s.Logout <- client
	s.mutex.Unlock()
}

func (s *Server) SendMessageToTransmit(message []byte) {
	s.mutex.Lock()
	s.Transmit <- message
	s.mutex.Unlock()
}

func (s *Server) RemoveClient(uuid string) {
	s.mutex.Lock()
	delete(s.Clients, uuid)
	s.mutex.Unlock()
}

// triggerPrivateAgent 私聊触发 Agent：调用 LLM 生成回复，持久化后推送给用户。
// 在协程中执行，不阻塞主转发循环。失败时返回降级提示。
func (s *Server) triggerPrivateAgent(userID string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(constants.AgentTimeoutSec)*time.Second)
	defer cancel()

	reply := agent.AgentService.TriggerPrivate(ctx, userID)
	if reply == nil || len(reply.Payload) == 0 {
		return
	}
	s.pushToUser(userID, reply.Payload, reply.Message.Uuid)
}

// triggerGroupAgent 群聊触发 Agent：调用 LLM 生成回复，持久化后群发给除 Agent 外的在线成员。
func (s *Server) triggerGroupAgent(groupID, userID, rawContent string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(constants.AgentTimeoutSec)*time.Second)
	defer cancel()

	reply := agent.AgentService.TriggerGroup(ctx, groupID, userID, rawContent)
	if reply == nil || len(reply.Payload) == 0 {
		return
	}
	s.pushToGroup(groupID, reply.Payload, reply.Message.Uuid)
}

// pushToUser 把 Agent 回复载荷推送给指定在线用户（参考现有私聊推送逻辑）。
func (s *Server) pushToUser(uuid string, payload []byte, msgUuid string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if c, ok := s.Clients[uuid]; ok {
		c.SendBack <- &MessageBack{Message: payload, Uuid: msgUuid}
	}
}

// pushToGroup 把 Agent 回复载荷群发给除 Agent 外的在线成员。
func (s *Server) pushToGroup(groupID string, payload []byte, msgUuid string) {
	var group model.GroupInfo
	if res := dao.GormDB.Where("uuid = ?", groupID).First(&group); res.Error != nil {
		zlog.Error("Agent pushToGroup load group: " + res.Error.Error())
		return
	}
	var members []string
	if err := json.Unmarshal(group.Members, &members); err != nil {
		zlog.Error("Agent pushToGroup unmarshal members: " + err.Error())
		return
	}
	mb := &MessageBack{Message: payload, Uuid: msgUuid}
	s.mutex.Lock()
	defer s.mutex.Unlock()
	for _, m := range members {
		if m == constants.AgentBotUuid {
			continue
		}
		if c, ok := s.Clients[m]; ok {
			c.SendBack <- mb
		}
	}
}
