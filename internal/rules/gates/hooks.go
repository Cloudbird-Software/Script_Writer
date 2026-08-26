package gates

import (
	"fmt"
	"strings"

	"github.com/Cloudbird-Software/Script_Writer/internal/canon"
	"github.com/Cloudbird-Software/Script_Writer/internal/rules"
	"github.com/Cloudbird-Software/Script_Writer/internal/state"
)

// QuoteGrounding 引文接地门（issue #1 §B-2 门 5）：任何"想起某人说过/当年的约定"，
// 引文必须能在前文语料中逐字或近似（相似度 ≥0.8）检索到出处。
// 拦：E30『过客有期』——全剧无人说过，结局高潮建立在不存在的台词上。
func QuoteGrounding(c *canon.Canon, eps []state.Episode, _ []state.Snapshot) []state.Violation {
	var vs []state.Violation
	triggers := []string{"想起", "记得", "回忆", "当年", "曾说", "说过", "那句话", "约定", "那句"}
	var prior strings.Builder // 归一化前文语料（逐集累积）
	for _, ep := range eps {
		sents := rules.Sentences(ep.Text)
		for si, sent := range sents {
			hasTrigger := false
			for _, tr := range triggers {
				if strings.Contains(sent, tr) {
					hasTrigger = true
					break
				}
			}
			if !hasTrigger {
				continue
			}
			sentRs := []rune(sent)
			for _, q := range rules.QuoteSpans(sentRs) {
				quote := string(sentRs[q.Open+1 : q.Close])
				if quote == "" || len([]rune(quote)) < 2 {
					continue
				}
				if !grounded(quote, prior.String()) {
					vs = append(vs, state.Violation{
						Gate: state.GateQuoteGrounding, Episode: ep.Ep,
						Position: fmt.Sprintf("第%d句", si+1),
						Expected: "前文出现过的原话（逐字或相似度 ≥0.8）",
						Actual:   quote, Severity: state.SeverityError,
						Message: "引文无出处（凭空引文，E30『过客有期』类缺陷）",
					})
				}
			}
		}
		prior.WriteString(rules.Normalize(ep.Text))
	}
	return vs
}

// grounded 判断引文 q（原文）是否能在归一化前文 corpus 中找到出处。
func grounded(q, corpus string) bool {
	if corpus == "" {
		return false
	}
	nq := rules.Normalize(q)
	if len([]rune(nq)) < 2 {
		return true // 过短无法判定，不拦
	}
	if strings.Contains(corpus, nq) {
		return true
	}
	// 近似：对 corpus 等长滑窗求最大相似度。
	rc := []rune(corpus)
	rq := []rune(nq)
	for w := len(rq) - 1; w <= len(rq)+1; w++ {
		if w <= 0 || w > len(rc) {
			continue
		}
		for s := 0; s+w <= len(rc); s++ {
			window := rc[s : s+w]
			d := rules.Lev(string(rq), string(window))
			sim := 1 - float64(d)/float64(max(len(rq), w))
			if sim >= 0.8 {
				return true
			}
		}
	}
	return false
}

// HookPayoff 钩子/回收门（issue #1 §B-2 门 7）：
// ① 每集必须有 hook 或 button（拦 9 集暖收）；② open loop 悬置 >6 集告警、>10 集硬失败；
// ③ P0 loop 必须在结局前全部 closed；④ 相邻集钩接：第 N 集 hook 的承接关键词必须
// 出现在第 N+1 集开头 300 字内（拦 E14→E15 因果断裂）。
func HookPayoff(c *canon.Canon, eps []state.Episode, snaps []state.Snapshot) []state.Violation {
	var vs []state.Violation
	for i, ep := range eps {
		if !ep.Finale && len(ep.Delta.HooksOpened) == 0 {
			vs = append(vs, state.Violation{
				Gate: state.GateHookPayoff, Episode: ep.Ep, Position: "delta.hooks_opened",
				Expected: "每集至少 1 个 hook 或 button", Actual: "0",
				Severity: state.SeverityError,
				Message:  "无钩子暖收（E4/E7/E13… 九集类缺陷：观众在此流失）",
			})
		}
		// 相邻集钩接（下一集开头 300 字承接）。
		if i+1 < len(eps) {
			opening := rules.FirstN(eps[i+1].Text, 300)
			for _, h := range ep.Delta.HooksOpened {
				if len(h.PickupKeywords) == 0 {
					vs = append(vs, state.Violation{
						Gate: state.GateHookPayoff, Episode: ep.Ep,
						Position: "delta.hooks_opened." + h.LoopID,
						Expected: "申报 pickup_keywords（供相邻集承接校验）", Actual: "空",
						Severity: state.SeverityWarn, Message: "钩子未申报承接关键词，无法校验钩接",
					})
					continue
				}
				if !rules.PickupKeywordHit(h.PickupKeywords, opening) {
					vs = append(vs, state.Violation{
						Gate: state.GateHookPayoff, Episode: ep.Ep,
						Position: "delta.hooks_opened." + h.LoopID,
						Expected: fmt.Sprintf("E%d 开头 300 字内出现 %v 之一", eps[i+1].Ep, h.PickupKeywords),
						Actual:   rules.FirstN(opening, 20) + "…",
						Severity: state.SeverityError,
						Message:  "相邻集钩子断裂（E14→E15 类因果断裂）",
					})
				}
			}
		}
		// 悬置时限 + P0 清账（基于本集后快照）。
		if i < len(snaps) {
			for _, lp := range snaps[i].Loops {
				if !lp.Open() {
					continue
				}
				age := ep.Ep - lp.OpenedEp
				if age > 10 {
					vs = append(vs, state.Violation{
						Gate: state.GateHookPayoff, Episode: ep.Ep,
						Position: "loop." + lp.LoopID,
						Expected: "悬置 ≤10 集", Actual: fmt.Sprintf("已悬置 %d 集（E%d 埋设）", age, lp.OpenedEp),
						Severity: state.SeverityError,
						Message:  "伏笔超期未回收（7 个断掉的伏笔类缺陷）",
					})
				} else if age > 6 {
					vs = append(vs, state.Violation{
						Gate: state.GateHookPayoff, Episode: ep.Ep,
						Position: "loop." + lp.LoopID,
						Expected: "悬置 ≤6 集", Actual: fmt.Sprintf("已悬置 %d 集", age),
						Severity: state.SeverityWarn, Message: "伏笔临近超期",
					})
				}
				if ep.Finale && lp.Priority == "P0" {
					vs = append(vs, state.Violation{
						Gate: state.GateHookPayoff, Episode: ep.Ep,
						Position: "loop." + lp.LoopID,
						Expected: "P0 伏笔结局前回收", Actual: "仍未回收",
						Severity: state.SeverityError,
						Message:  "P0 伏笔未清账即结局（issue #1 §B-2 门 7-③）",
					})
				}
			}
		}
	}
	return vs
}
