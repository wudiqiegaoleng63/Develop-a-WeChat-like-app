package agent

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/schema"

	"kama-chat-server/pkg/constants"
	"kama-chat-server/pkg/zlog"
)

// ErrNotImplemented 流式接口暂未实现，MVP 阶段预留。
var ErrNotImplemented = errors.New("stream chat not implemented in MVP")

// Message 角色
const (
	RoleSystem    = "system"
	RoleUser      = "user"
	RoleAssistant = "assistant"
)

// Message LLM 调用的单条消息
type Message struct {
	Role    string
	Content string
}

// AgentClient 抽象 LLM 调用，便于 Mock 与真实 Provider 切换。
type AgentClient interface {
	// Generate 非流式生成回复
	Generate(ctx context.Context, messages []Message) (string, error)
	// Stream 流式生成（MVP 预留，暂返回 ErrNotImplemented）
	Stream(ctx context.Context, messages []Message) (<-chan string, error)
}

// DefaultClient 全局单例，在 init 中根据 LLM_PROVIDER 环境变量决定实现。
var DefaultClient AgentClient

func init() {
	DefaultClient = newClientFromEnv()
}

// newClientFromEnv 根据 LLM_PROVIDER 选择实现：
//   - "mock" 或未设置 → MockClient（本地无 key 调试）
//   - "openai"/"eino"  → EinoClient（真实 LLM）
func newClientFromEnv() AgentClient {
	provider := os.Getenv("LLM_PROVIDER")
	switch provider {
	case "openai", "eino":
		c, err := NewEinoClient()
		if err != nil {
			zlog.Error("Agent: EinoClient 初始化失败，回退到 MockClient: " + err.Error())
			return NewMockClient()
		}
		zlog.Info("Agent: 使用 EinoClient provider=" + provider + " model=" + os.Getenv("LLM_MODEL"))
		return c
	default:
		zlog.Info("Agent: 使用 MockClient（设置 LLM_PROVIDER=openai 启用真实 LLM）")
		return NewMockClient()
	}
}

// ===================== MockClient =====================

// MockClient 不依赖外部 LLM，用于本地开发与单元测试。
type MockClient struct{}

func NewMockClient() *MockClient { return &MockClient{} }

func (m *MockClient) Generate(ctx context.Context, messages []Message) (string, error) {
	// 模拟一点延迟，便于测试前端体验
	select {
	case <-time.After(300 * time.Millisecond):
	case <-ctx.Done():
		return "", ctx.Err()
	}
	if len(messages) == 0 {
		return "你好，我是 AI助手，有什么可以帮你？", nil
	}
	last := messages[len(messages)-1].Content
	return fmt.Sprintf("[Mock] 收到你的消息「%s」。我是 AI助手，配置 LLM_PROVIDER=openai 并设置 OPENAI_API_KEY 后可接入真实大模型。", last), nil
}

func (m *MockClient) Stream(ctx context.Context, messages []Message) (<-chan string, error) {
	return nil, ErrNotImplemented
}

// ===================== EinoClient =====================

// EinoClient 基于 CloudWeGo Eino 的 openai ChatModel 实现。
// 支持 OpenAI 官方及任何 OpenAI 兼容接口（DeepSeek/Qwen/Moonshot 等）。
type EinoClient struct {
	chatModel *openai.ChatModel
}

// NewEinoClient 从环境变量构造 EinoClient。
// 必填：OPENAI_API_KEY、LLM_MODEL；可选：OPENAI_BASE_URL。
func NewEinoClient() (*EinoClient, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	modelName := os.Getenv("LLM_MODEL")
	baseURL := os.Getenv("OPENAI_BASE_URL")
	if apiKey == "" || modelName == "" {
		return nil, errors.New("OPENAI_API_KEY 和 LLM_MODEL 不能为空")
	}

	chatModel, err := openai.NewChatModel(context.Background(), &openai.ChatModelConfig{
		APIKey:  apiKey,
		Model:   modelName,
		BaseURL: baseURL,
	})
	if err != nil {
		return nil, fmt.Errorf("NewChatModel: %w", err)
	}
	return &EinoClient{chatModel: chatModel}, nil
}

func (e *EinoClient) Generate(ctx context.Context, messages []Message) (string, error) {
	schemaMsgs := make([]*schema.Message, 0, len(messages))
	for _, m := range messages {
		role := schema.User
		switch m.Role {
		case RoleSystem:
			role = schema.System
		case RoleAssistant:
			role = schema.Assistant
		}
		schemaMsgs = append(schemaMsgs, &schema.Message{Role: role, Content: m.Content})
	}

	ctx, cancel := context.WithTimeout(ctx, time.Duration(constants.AgentTimeoutSec)*time.Second)
	defer cancel()

	resp, err := e.chatModel.Generate(ctx, schemaMsgs)
	if err != nil {
		return "", err
	}
	return resp.Content, nil
}

func (e *EinoClient) Stream(ctx context.Context, messages []Message) (<-chan string, error) {
	return nil, ErrNotImplemented
}
