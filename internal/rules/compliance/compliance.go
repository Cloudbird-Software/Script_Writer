// Package compliance 是 M4 品牌安全门（issue #1 §B-2 门 11）。
//
// 分级词表扫描：
//   - hard  硬失败：官商往来/权力寻租（铺条官路）、特权炫耀（有令牌才算有脸面）、
//     迷信（妖火/摄魂）——真实品牌不能背的叙事线
//   - flag  标记人审：绝对化用语（头一份/汴京第一）——进风险清单，
//     剪宣传物料前必须过一遍
//
// 词表是资产（随剧目/客户变化），进 canon config.compliance，缺省词表直接
// 取材 issue #1 已发生缺陷（E14/E26/E27/E28 全套）。
//
// 深接口纪律：本包只导出 Rule() 一个构造器。
package compliance

import (
	"fmt"
	"strings"

	"github.com/Cloudbird-Software/Script_Writer/internal/canon"
	"github.com/Cloudbird-Software/Script_Writer/internal/rules"
	"github.com/Cloudbird-Software/Script_Writer/internal/state"
)

// Rule 返回品牌安全门禁实例。
func Rule() rules.Rule { return rule{} }

type rule struct{}

// ID 实现 rules.Rule。
func (rule) ID() string { return state.GateCompliance }

// Check 实现 rules.Rule：逐句扫分级词表。
func (rule) Check(in rules.Input) []state.Violation {
	var vs []state.Violation
	cats := in.Canon.Config.WithDefaults().Compliance.Categories
	for _, ep := range in.Episodes {
		for si, sent := range rules.Sentences(ep.Text) {
			pos := fmt.Sprintf("第%d句", si+1)
			for _, cat := range cats {
				for _, p := range cat.Patterns {
					if !strings.Contains(sent, p) {
						continue
					}
					v := state.Violation{
						Gate: state.GateCompliance, Episode: ep.Ep, Position: pos,
						Expected: "品牌安全红线外表述", Actual: p,
						Message: fmt.Sprintf("合规风险 [%s]：%q（人审清单）", cat.ID, p),
					}
					if cat.Level == canon.ComplianceHard {
						v.Severity = state.SeverityError
						v.Message = fmt.Sprintf("合规红线 [%s]：%q（真实品牌不能背的叙事）", cat.ID, p)
					} else {
						v.Severity = state.SeverityWarn
					}
					vs = append(vs, v)
				}
			}
		}
	}
	return vs
}
