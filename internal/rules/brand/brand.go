// Package brand 是 M4 营销门（issue #1 §B-2 门 10）。
//
// 拦：
//   - 排期集未申报卖点（排期表是营销侧的硬约束，申报缺位 = 该集白卖）
//   - 会员体系（令牌）材质漂移——"令牌材质四套并存"类缺陷：
//     牌类道具的材质字必须在该道具的 tiers（+ visual_rule 材质字）内
//
// 排期一致性（申报≠排期）、重复维度、服务占比、每卖点集数上限已分别由
// state 台账层与 canon 结构校验负责，本门不重复。
//
// 深接口纪律：本包只导出 Rule() 一个构造器。
package brand

import (
	"fmt"
	"strings"

	"github.com/Cloudbird-Software/Script_Writer/internal/canon"
	"github.com/Cloudbird-Software/Script_Writer/internal/rules"
	"github.com/Cloudbird-Software/Script_Writer/internal/state"
)

// Rule 返回营销门禁实例。
func Rule() rules.Rule { return rule{} }

type rule struct{}

// ID 实现 rules.Rule。
func (rule) ID() string { return state.GateBrand }

// 全局材质字表（牌类道具材质漂移检测用）。
var materialChars = map[rune]bool{
	'木': true, '铜': true, '银': true, '金': true, '玉': true,
	'铁': true, '竹': true, '骨': true, '角': true,
}

// Check 实现 rules.Rule。
func (rule) Check(in rules.Input) []state.Violation {
	var vs []state.Violation

	// 1. 排期覆盖的每集必须申报主卖点（缺申报 = 该集白卖）。
	scheduled := map[int]bool{}
	for _, s := range in.Canon.SellingPoints.Schedule {
		scheduled[s.Ep] = true
	}
	for _, ep := range in.Episodes {
		if scheduled[ep.Ep] && ep.Delta.SellingPoint == "" {
			vs = append(vs, state.Violation{
				Gate: state.GateBrand, Episode: ep.Ep, Position: "delta.selling_point",
				Expected: "按排期表申报本集主卖点", Actual: "空",
				Severity: state.SeverityError,
				Message:  "排期集未申报卖点（营销侧硬约束：每集主卖点一对一排期）",
			})
		}
	}

	// 2. 牌类道具材质漂移（会员体系一致性）：句中"材质字+牌"（含 铜令牌 型
	//    隔字组合）的材质必须 ∈ 某个牌类道具的合法材质集。
	legal := legalMaterials(in.Canon) // 材质字 → 合法
	if len(legal) > 0 {
		for _, ep := range in.Episodes {
			for si, sent := range rules.Sentences(ep.Text) {
				rs := []rune(sent)
				for i := 0; i < len(rs); i++ {
					if !materialChars[rs[i]] {
						continue
					}
					if !boardNearby(rs, i) {
						continue
					}
					if !legal[rs[i]] {
						vs = append(vs, state.Violation{
							Gate: state.GateBrand, Episode: ep.Ep,
							Position: fmt.Sprintf("第%d句", si+1),
							Expected: "tiers 内材质（材质必须全剧唯一）",
							Actual:   string(rs[i]) + "牌",
							Severity: state.SeverityError,
							Message:  "会员体系材质漂移（issue #1：令牌材质四套并存，会员规则自相矛盾）",
						})
					}
				}
			}
		}
	}
	return vs
}

// boardNearby 判断 rs[i] 之后 0~2 个 rune 内是否出现"牌"（覆盖 木牌/铜令牌 组合）。
func boardNearby(rs []rune, i int) bool {
	for j := i + 1; j <= i+2 && j < len(rs); j++ {
		if rs[j] == '牌' {
			return true
		}
	}
	return false
}

// legalMaterials 汇总全部牌类道具（name 以"牌"结尾）的合法材质字：tiers ∪ visual_rule 材质字。
func legalMaterials(c *canon.Canon) map[rune]bool {
	out := map[rune]bool{}
	for _, p := range c.Props {
		if !strings.HasSuffix(p.Name, "牌") {
			continue
		}
		for _, t := range p.Tiers {
			for _, r := range t {
				if materialChars[r] {
					out[r] = true
				}
			}
		}
		for _, r := range p.VisualRule {
			if materialChars[r] {
				out[r] = true
			}
		}
	}
	return out
}
