package engine

import (
	"fmt"
	"strings"

	"github.com/Cloudbird-Software/Script_Writer/internal/canon"
	"github.com/Cloudbird-Software/Script_Writer/internal/rules"
	"github.com/Cloudbird-Software/Script_Writer/internal/state"
)

// Linkage 重跑 ±1 集联动校验（issue #1 §D-2）：
// 任何一集被重跑，必须同时校验"前一集结尾钩子 → 本集开头承接"与
// "本集结尾钩子 → 后一集开头承接"——E14 重跑后钩子换了、E15 还是旧的，就是缺这道。
//
// rerunEp 为被重跑的集号；eps 中该集正文应为新文本。
func Linkage(c *canon.Canon, eps []state.Episode, rerunEp int) []state.Violation {
	idx := -1
	for i, ep := range eps {
		if ep.Ep == rerunEp {
			idx = i
			break
		}
	}
	if idx < 0 {
		return []state.Violation{{
			Gate: "linkage", Episode: rerunEp, Severity: state.SeverityError,
			Message: fmt.Sprintf("E%d 不在 episodes 中", rerunEp),
		}}
	}
	var vs []state.Violation
	cur := eps[idx]
	// 关节 1：前一集钩子 → 本集（新文本）开头 300 字承接。
	if idx > 0 {
		prev := eps[idx-1]
		vs = append(vs, checkJoints(prev, cur)...)
	}
	// 关节 2：本集（新文本）钩子 → 后一集开头 300 字承接。
	if idx+1 < len(eps) {
		next := eps[idx+1]
		vs = append(vs, checkJoints(cur, next)...)
	}
	return vs
}

// checkJoints 校验 from 集的钩子在 to 集开头 300 字内被承接。
func checkJoints(from, to state.Episode) []state.Violation {
	var vs []state.Violation
	opening := rules.FirstN(to.Text, 300)
	for _, h := range from.Delta.HooksOpened {
		if len(h.PickupKeywords) == 0 {
			vs = append(vs, state.Violation{
				Gate: "linkage", Episode: to.Ep,
				Position: "loop." + h.LoopID,
				Expected: "申报 pickup_keywords", Actual: "空",
				Severity: state.SeverityWarn,
				Message:  fmt.Sprintf("E%d 钩子 %s 未申报承接关键词", from.Ep, h.LoopID),
			})
			continue
		}
		found := false
		for _, kw := range h.PickupKeywords {
			if kw != "" && strings.Contains(opening, kw) {
				found = true
				break
			}
		}
		if !found {
			vs = append(vs, state.Violation{
				Gate: "linkage", Episode: to.Ep,
				Position: "loop." + h.LoopID,
				Expected: fmt.Sprintf("E%d 开头 300 字内出现 %v 之一", to.Ep, h.PickupKeywords),
				Actual:   rules.FirstN(opening, 20) + "…",
				Severity: state.SeverityError,
				Message:  fmt.Sprintf("重跑 ±1 联动断裂：E%d 钩子未被 E%d 承接（E14→E15 类缺陷）", from.Ep, to.Ep),
			})
		}
	}
	return vs
}
