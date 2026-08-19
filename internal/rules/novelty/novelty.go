// Package novelty 是 M4 新鲜度门（issue #1 §B-2 门 8，规则版）。
//
// 拦"30 集同一套路"：
//   - 每集必须申报 ≥min_new_facts 条 new_fact 与 ≥min_state_changes 条
//     state_change（没有新事实/新状态的一集 = 复述集）
//   - 与前 compare_window_eps 集做字符 n-gram 重复度：本集 n-gram 中出现
//     在前文窗口里的比例 > max_repeat_ratio 即硬失败（拦 E19/E21/E29 三集复述、
//     同一"惊叹模板"连播）
//
// embedding 相似度（issue #1 原文）属 M5 LLM 旁路，规则版以 n-gram 覆盖。
//
// 深接口纪律：本包只导出 Rule() 一个构造器。
package novelty

import (
	"fmt"
	"strings"

	"github.com/Cloudbird-Software/Script_Writer/internal/rules"
	"github.com/Cloudbird-Software/Script_Writer/internal/state"
)

// ngramLen 是重复度检测的字符 n-gram 长度（中文 4 字窗：短语级复现）。
const ngramLen = 4

// Rule 返回新鲜度门禁实例。
func Rule() rules.Rule { return rule{} }

type rule struct{}

// ID 实现 rules.Rule。
func (rule) ID() string { return state.GateNovelty }

// Check 实现 rules.Rule。
func (rule) Check(in rules.Input) []state.Violation {
	var vs []state.Violation
	cfg := in.Canon.Config.WithDefaults().Novelty

	for i, ep := range in.Episodes {
		// 1. 申报下限：new_fact / state_change。
		if n := len(ep.Delta.NewFacts); n < cfg.MinNewFacts {
			vs = append(vs, state.Violation{
				Gate: state.GateNovelty, Episode: ep.Ep, Position: "delta.new_facts",
				Expected: fmt.Sprintf("每集申报 ≥%d 条新事实", cfg.MinNewFacts),
				Actual:   fmt.Sprint(n), Severity: state.SeverityError,
				Message: "未申报新事实（没有新事实的一集是复述集，E19/E21/E29 类缺陷）",
			})
		}
		if n := len(ep.Delta.StateChanges); n < cfg.MinStateChanges {
			vs = append(vs, state.Violation{
				Gate: state.GateNovelty, Episode: ep.Ep, Position: "delta.state_changes",
				Expected: fmt.Sprintf("每集申报 ≥%d 条状态变化", cfg.MinStateChanges),
				Actual:   fmt.Sprint(n), Severity: state.SeverityError,
				Message: "未申报状态变化（世界没有往前走的一集是原地踏步）",
			})
		}

		// 2. 与前 N 集的 n-gram 重复度（首集无前文，跳过）。
		if i == 0 {
			continue
		}
		lo := i - cfg.CompareWindowEps
		if lo < 0 {
			lo = 0
		}
		var prior strings.Builder
		for j := lo; j < i; j++ {
			prior.WriteString(rules.Normalize(in.Episodes[j].Text))
		}
		if ratio, grams := repeatRatio(ep.Text, prior.String()); grams > 0 && ratio > cfg.MaxRepeatRatio {
			vs = append(vs, state.Violation{
				Gate: state.GateNovelty, Episode: ep.Ep, Position: "正文",
				Expected: fmt.Sprintf("与前 %d 集重复度 ≤%.0f%%", cfg.CompareWindowEps, cfg.MaxRepeatRatio*100),
				Actual:   fmt.Sprintf("%.0f%%（%d 组 %d-gram 中重复）", ratio*100, grams, ngramLen),
				Severity: state.SeverityError,
				Message:  "复述前文（同一套路连播；embedding 深查属 M5，规则版以 n-gram 拦）",
			})
		}
	}
	return vs
}

// repeatRatio 返回本集 4-gram 中出现在 prior 里的比例与 gram 总数。
func repeatRatio(text, prior string) (float64, int) {
	if prior == "" {
		return 0, 0
	}
	n := rules.Normalize(text)
	rs := []rune(n)
	if len(rs) < ngramLen {
		return 0, 0
	}
	total, hit := 0, 0
	for i := 0; i+ngramLen <= len(rs); i++ {
		total++
		if strings.Contains(prior, string(rs[i:i+ngramLen])) {
			hit++
		}
	}
	if total == 0 {
		return 0, 0
	}
	return float64(hit) / float64(total), total
}
