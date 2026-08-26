// Package voice 是 M4 声音指纹门（issue #1 §B-2 软门 13）。
//
// 拦"所有人说话一个味儿"：按角色台词做句长指纹——引号台词按
// 规则版归属（说话语境中唯一具名角色，rules.SoleCharacter）分桶，
// 各角色平均台词长度的标准差 < min_profile_spread 即 warn。
// 当前富商、捕快、小吏、绣娘的台词风格几乎无差别，正是这道门要暴露的。
//
// 文白比/口头禅指纹与代词消解属 M5 LLM 旁路（BAML prompt 版本化）。
//
// 深接口纪律：本包只导出 Rule() 一个构造器。
package voice

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/Cloudbird-Software/Script_Writer/internal/rules"
	"github.com/Cloudbird-Software/Script_Writer/internal/state"
)

// Rule 返回声音指纹门禁实例。
func Rule() rules.Rule { return rule{} }

type rule struct{}

// ID 实现 rules.Rule。
func (rule) ID() string { return state.GateVoice }

// Check 实现 rules.Rule。
func (rule) Check(in rules.Input) []state.Violation {
	cfg := in.Canon.Config.WithDefaults().Voice

	// 1. 归属收集：引号台词 → 说话语境中唯一具名角色。
	lengths := map[string][]int{}
	for _, ep := range in.Episodes {
		rs := []rune(ep.Text)
		for _, q := range rules.QuoteSpans(rs) {
			ctxStart := contextStart(rs, q.Open)
			if id, _ := rules.SoleCharacter(in.Canon, string(rs[ctxStart:q.Open])); id != "" {
				lengths[id] = append(lengths[id], q.Close-q.Open-1) // 引号内容 rune 数
			}
		}
	}

	// 2. 指纹：各角色平均台词长度的标准差（≥2 条台词的角色才入样）。
	ids := make([]string, 0, len(lengths))
	for id, ls := range lengths {
		if len(ls) >= 2 {
			ids = append(ids, id)
		}
	}
	if len(ids) < 2 {
		return nil // 样本不足，不判（单人剧/旁白剧误报）
	}
	sort.Strings(ids)
	avgs := make([]float64, len(ids))
	for i, id := range ids {
		sum := 0
		for _, l := range lengths[id] {
			sum += l
		}
		avgs[i] = float64(sum) / float64(len(lengths[id]))
	}
	sd := stddev(avgs)
	if sd < cfg.MinProfileSpread {
		var prof strings.Builder
		for i, id := range ids {
			if e, ok := in.Canon.EntityByID(id); ok {
				id = e.CanonicalName
			}
			prof.WriteString(fmt.Sprintf("%s≈%.1f字 ", id, avgs[i]))
		}
		return []state.Violation{{
			Gate: state.GateVoice, Episode: in.Episodes[len(in.Episodes)-1].Ep,
			Position: "台词指纹",
			Expected: fmt.Sprintf("角色间平均句长标准差 ≥%.1f 字", cfg.MinProfileSpread),
			Actual:   fmt.Sprintf("%.1f（%s）", sd, strings.TrimSpace(prof.String())),
			Severity: state.SeverityWarn,
			Message:  "所有人说话一个味儿（台词指纹离散度不足；文白比/口头禅深查属 M5）",
		}}
	}
	return nil
}

// contextStart 返回开口引号之前的最近句读边界（含）之后的位置：
// 说话语境 = 上一个句读到开口引号之间的文本。
func contextStart(rs []rune, open int) int {
	for i := open - 1; i > 0; i-- {
		if rules.IsSentenceBreak(rs[i]) {
			return i + 1
		}
	}
	return 0
}

func stddev(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	var sum float64
	for _, x := range xs {
		sum += x
	}
	mean := sum / float64(len(xs))
	var sq float64
	for _, x := range xs {
		d := x - mean
		sq += d * d
	}
	return math.Sqrt(sq / float64(len(xs)))
}
