package engine

import (
	"github.com/Cloudbird-Software/Script_Writer/internal/rules"
	"github.com/Cloudbird-Software/Script_Writer/internal/rules/gates"
	"github.com/Cloudbird-Software/Script_Writer/internal/rules/hygiene"
)

// allRules 返回全部门禁（注册表）。新增门禁只改这里与对应 rules 子包——
// 门禁本体对 engine 而言只有一个 Rule 接口面（深接口纪律）。
func allRules() []rules.Rule {
	return []rules.Rule{
		// M2 五道硬门（issue #1 §B-2 门 1/3/4/5/7）。
		rules.FuncRule{Gate: "format", Fn: gates.Format},
		rules.FuncRule{Gate: "consistency", Fn: gates.Consistency},
		rules.FuncRule{Gate: "relationship", Fn: gates.Relationship},
		rules.FuncRule{Gate: "quote-grounding", Fn: gates.QuoteGrounding},
		rules.FuncRule{Gate: "hook-payoff", Fn: gates.HookPayoff},
		// M4 各域门禁（issue #1 §B-2 门 2/6/8~14）逐门接入。
		hygiene.Rule(), // 门 2：文本卫生（错别字/乱码/生僻字）
	}
}
