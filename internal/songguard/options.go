package songguard

// options 承载门面可配置项（当前为空实现占位；M5 旁路客户端接入时扩展）。
type options struct{}

// Option 是门面的配置函数（WithXxx 形式）。
type Option func(*options)

// defaultOptions 返回默认配置。
func defaultOptions() options { return options{} }
