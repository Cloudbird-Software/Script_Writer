package engine

import (
	"github.com/Cloudbird-Software/Script_Writer/internal/rules"
	"github.com/Cloudbird-Software/Script_Writer/internal/rules/arc"
	"github.com/Cloudbird-Software/Script_Writer/internal/rules/brand"
	"github.com/Cloudbird-Software/Script_Writer/internal/rules/compliance"
	"github.com/Cloudbird-Software/Script_Writer/internal/rules/emotion"
	"github.com/Cloudbird-Software/Script_Writer/internal/rules/gates"
	"github.com/Cloudbird-Software/Script_Writer/internal/rules/hygiene"
	"github.com/Cloudbird-Software/Script_Writer/internal/rules/lineownership"
	"github.com/Cloudbird-Software/Script_Writer/internal/rules/novelty"
	"github.com/Cloudbird-Software/Script_Writer/internal/rules/producibility"
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
		hygiene.Rule(),       // 门 2：文本卫生（错别字/乱码/生僻字）
		lineownership.Rule(), // 门 6：台词归属（slogan 塞错嘴/用量申报不符）
		emotion.Rule(),       // 软门 14：情绪曲线（连续同类型即 fail）
		arc.Rule(),           // 门 9：弧线（起点/升速/代价）
		brand.Rule(),         // 门 10：营销（排期申报/令牌材质漂移）
		compliance.Rule(),    // 门 11：品牌安全（分级词表 hard/flag）
		producibility.Rule(), // 门 12：可拍性（角色/场景/上屏/镜头/群演/成本标签）
		novelty.Rule(),       // 门 8：新鲜度（申报下限 + n-gram 重复度）
	}
}
