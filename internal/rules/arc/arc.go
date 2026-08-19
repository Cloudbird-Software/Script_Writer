// Package arc 是 M4 弧线门（issue #1 §B-2 门 9）。
//
// 拦：女主"权限等级"0→max_level 的成长弧线被写崩——
// 开局即内部人（E1 兜里已揣令牌）、E7 就当老板、E30 高潮不成立。
//
// 台账层（state）已管：单调不倒退、单步 ≤+1。本门补齐资产阈值侧：
//   - 首次申报必须从 0 起步（拦"开局即内部人"）
//   - level ≤ max_level
//   - 升速：距上次等级变更 ≥ min_eps_per_step 集（0→5 级即每 5 集最多 +1，
//     E5 升 1、E10 升 2……E25 升 5，把高潮留在结局）
//   - 升级必须填写 cost（她付出了什么——没有代价的升级不产生情绪）
//   - 结局集 level 仍为 0 → warn（30 集零成长）
//
// 深接口纪律：本包只导出 Rule() 一个构造器。
package arc

import (
	"fmt"

	"github.com/Cloudbird-Software/Script_Writer/internal/rules"
	"github.com/Cloudbird-Software/Script_Writer/internal/state"
)

// Rule 返回弧线门禁实例。
func Rule() rules.Rule { return rule{} }

type rule struct{}

// ID 实现 rules.Rule。
func (rule) ID() string { return state.GateArc }

// Check 实现 rules.Rule。
func (rule) Check(in rules.Input) []state.Violation {
	var vs []state.Violation
	cfg := in.Canon.Config.WithDefaults().Arc

	prevLevel := -1 // 上一次申报的等级（-1 = 尚未申报）
	prevSetEp := 0  // 最近一次等级变更所在集（0 = 基线前，首升按全集起算）
	first := true   // 是否首次申报
	for _, ep := range in.Episodes {
		a := ep.Delta.Arc
		if a == nil {
			continue
		}
		if first {
			first = false
			if a.Level != 0 {
				vs = append(vs, state.Violation{
					Gate: state.GateArc, Episode: ep.Ep, Position: "delta.arc.level",
					Expected: "0（弧线从零起步）", Actual: fmt.Sprint(a.Level),
					Severity: state.SeverityError,
					Message:  "开局即内部人（issue #1：E1 她已在柜台后、兜里已揣令牌——高潮因此不成立）",
				})
			}
		}
		if a.Level > cfg.MaxLevel {
			vs = append(vs, state.Violation{
				Gate: state.GateArc, Episode: ep.Ep, Position: "delta.arc.level",
				Expected: fmt.Sprintf("≤%d（max_level）", cfg.MaxLevel), Actual: fmt.Sprint(a.Level),
				Severity: state.SeverityError, Message: "主角权限等级越上限",
			})
		}
		if prevLevel >= 0 && a.Level > prevLevel {
			// 升级：必须有代价、必须有间隔（升速）。
			if a.Cost == "" {
				vs = append(vs, state.Violation{
					Gate: state.GateArc, Episode: ep.Ep, Position: "delta.arc.cost",
					Expected: "升级必须填写 cost（她付出了什么）", Actual: "空",
					Severity: state.SeverityError,
					Message:  "无代价的升级（E30 应是『她终于付出了一次代价』，而不是钥匙摆到面前）",
				})
			}
			if gap := ep.Ep - prevSetEp; gap < cfg.MinEpsPerStep {
				vs = append(vs, state.Violation{
					Gate: state.GateArc, Episode: ep.Ep, Position: "delta.arc.level",
					Expected: fmt.Sprintf("距上次变更 ≥%d 集（每 %d 集最多 +1）", cfg.MinEpsPerStep, cfg.MinEpsPerStep),
					Actual:   fmt.Sprintf("间隔 %d 集（上次变更 E%d）", gap, prevSetEp),
					Severity: state.SeverityError,
					Message:  "升级过快（E7 就当老板类：到结局她已当了 23 集掌柜，高潮完全不产生情绪）",
				})
			}
		}
		if prevLevel >= 0 && a.Level != prevLevel {
			prevSetEp = ep.Ep
		}
		prevLevel = a.Level
		// 结局集零成长 = 30 集白讲。
		if ep.Finale && a.Level == 0 {
			vs = append(vs, state.Violation{
				Gate: state.GateArc, Episode: ep.Ep, Position: "delta.arc.level",
				Expected: ">0（结局前弧线应有成长）", Actual: "0",
				Severity: state.SeverityWarn, Message: "结局零成长（全剧无弧线）",
			})
		}
	}
	return vs
}
