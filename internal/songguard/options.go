package songguard

// options 承载门面可配置项（深接口：调用方只看到 Option，不触内层包）。
type options struct {
	sidecarURL string // M5 LLM 旁路地址；空 = 不跑 LLM pass（默认）
	provider   string // 透传给 sidecar 的 provider 覆盖（mock | openai | ""）
}

// Option 是门面的配置函数（WithXxx 形式）。
type Option func(*options)

// WithSidecar 启用 M5 LLM 软 pass：规则门禁之后追加语义兜底
// （sweep 巡检 + reader 观众模拟），结论一律 warn 级建议，永不阻断交付。
// baseURL 形如 http://127.0.0.1:8710（sidecar 唯一端点 /v1/llm-check）。
// sidecar 不可用时降级为单条可见 warn，主流程不受影响。
func WithSidecar(baseURL string) Option {
	return func(o *options) { o.sidecarURL = baseURL }
}

// WithProvider 覆盖 sidecar 的 LLM provider（默认由 sidecar 决定，通常 mock）。
// 须与 WithSidecar 搭配；未启用旁路时无效果。
func WithProvider(name string) Option {
	return func(o *options) { o.provider = name }
}

// defaultOptions 返回默认配置（不跑 LLM pass）。
func defaultOptions() options { return options{} }
