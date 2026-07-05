package constants

// Agent 相关常量。集中管理 AI 助手在系统中的身份与行为参数，
// 便于后续调整而不必散落修改多处代码。
const (
	// AgentBotUuid AI 助手作为系统用户的固定 UUID。
	// 以 'U' 开头以兼容现有 message dispatch 的 ReceiveId[0]=='U' 私聊分支。
	// 长度 20 字符（U + 13 个 0 + AGENT0），与 user_info.uuid 的 char(20) 列一致。
	AgentBotUuid = "U0000000000000AGENT0"

	// AgentBotNickname AI 助手展示昵称
	AgentBotNickname = "AI助手"

	// AgentBotAvatar AI 助手默认头像（与注册用户默认头像一致）
	AgentBotAvatar = "https://cube.elemecdn.com/0/88/03b0d39583f48206768a7534e55bcpng.png"

	// AgentPrivateContextLen 私聊上下文历史消息条数
	AgentPrivateContextLen = 10

	// AgentGroupContextLen 群聊上下文历史消息条数
	AgentGroupContextLen = 20

	// AgentTimeoutSec 单次 LLM 调用超时秒数
	AgentTimeoutSec = 25

	// AgentMaxInputLen 用户输入最大字符数，超长截断
	AgentMaxInputLen = 4000

	// AgentErrMsg LLM 调用失败时返回给用户的降级文案
	AgentErrMsg = "AI助手暂时不可用，请稍后再试"
)

// AgentGroupTriggers 群聊触发 Agent 的关键词（大小写不敏感）。
// 命中其中任一前缀即触发，触发词会被剥离后再传给 LLM。
var AgentGroupTriggers = []string{
	"@AI助手",
	"@agent",
	"/ai ",
	"/ai",
}
