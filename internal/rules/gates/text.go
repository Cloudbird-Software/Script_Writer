// Package gates 保留 M2 五道硬门（issue #1 §B-2）中的文本三门的纯函数实现：
//
//	Format         格式门：字数区间 ±10%、全半角统一、人名禁非中文字符、无 markdown 残留
//	Consistency    一致性门：forbidden_names 精确比对 + 称谓/地名后缀漂移检测
//	Relationship   关系门：已有相遇记录的角色禁止"初次相识"叙事
//
// M4 各域门禁按深接口纪律逐个独立成 rules/<gate> 子包；本包不再扩容。
package gates

import (
	"fmt"
	"strings"

	"github.com/Cloudbird-Software/Script_Writer/internal/canon"
	"github.com/Cloudbird-Software/Script_Writer/internal/rules"
	"github.com/Cloudbird-Software/Script_Writer/internal/state"
)

// Format 格式门（issue #1 §B-2 门 1）：字数区间 ±10%、全半角统一、
// 人名禁非中文字符（单字母混排如 A福）、无 markdown 残留。失败处理=自动修复，
// 本工具只报告不改写。
func Format(c *canon.Canon, eps []state.Episode, _ []state.Snapshot) []state.Violation {
	var vs []state.Violation
	for _, ep := range eps {
		n := rules.CountChars(ep.Text)
		if ep.TargetChars > 0 {
			lo := float64(ep.TargetChars) * 0.9
			hi := float64(ep.TargetChars) * 1.1
			if float64(n) < lo || float64(n) > hi {
				vs = append(vs, state.Violation{
					Gate: state.GateFormat, Episode: ep.Ep, Position: "字数",
					Expected: fmt.Sprintf("[%d, %d]（目标 %d ±10%%）", int(lo), int(hi), ep.TargetChars),
					Actual:   fmt.Sprint(n), Severity: state.SeverityError,
					Message: fmt.Sprintf("字数越界（issue #1：实测 270–700 字 2.5 倍方差）"),
				})
			}
		}
		if f, h, mixed := rules.PunctMix(ep.Text); mixed {
			vs = append(vs, state.Violation{
				Gate: state.GateFormat, Episode: ep.Ep, Position: "标点",
				Expected: "全半角统一（同一交付物一种字符集）",
				Actual:   fmt.Sprintf("全角 %q 与半角 %q 并存", f, h),
				Severity: state.SeverityError, Message: "全半角标点混用（E3/E17/E21/E24 类缺陷）",
			})
		}
		for _, m := range rules.MixedScriptNames(ep.Text) {
			vs = append(vs, state.Violation{
				Gate: state.GateFormat, Episode: ep.Ep, Position: "混排名",
				Expected: "纯中文名或登记别名", Actual: m,
				Severity: state.SeverityError, Message: "单字母与汉字混排的疑似人名（E9 A福 类）",
			})
		}
		for _, md := range rules.MarkdownResidue(ep.Text) {
			vs = append(vs, state.Violation{
				Gate: state.GateFormat, Episode: ep.Ep, Position: "markdown",
				Expected: "交付正文无 markdown 残留", Actual: md,
				Severity: state.SeverityError, Message: "markdown 残留",
			})
		}
	}
	return vs
}

// Consistency 一致性门（issue #1 §B-2 门 3）：forbidden_names 精确比对 +
// 称谓后缀（X捕快/X掌柜…）与地名后缀（X驿站…）漂移检测。
// 拦：渔/宁漂移、宋驿/靖康驿站、令牌材质四变的名称面。
func Consistency(c *canon.Canon, eps []state.Episode, _ []state.Snapshot) []state.Violation {
	var vs []state.Violation
	known := c.KnownNames() // name → entity id
	for _, ep := range eps {
		for si, sent := range rules.Sentences(ep.Text) {
			pos := fmt.Sprintf("第%d句", si+1)
			for _, e := range c.Entities {
				for _, f := range e.ForbiddenNames {
					if strings.Contains(sent, f) {
						vs = append(vs, state.Violation{
							Gate: state.GateConsistency, Episode: ep.Ep, Position: pos,
							Expected: e.CanonicalName, Actual: f,
							Severity: state.SeverityError,
							Message:  fmt.Sprintf("黑名单名称（实体 %s 禁用写法）", e.ID),
						})
					}
				}
			}
			// 称谓漂移：X+称谓 不在白名单，但与某已知名距离 ≤1 → 硬失败。
			for _, title := range c.Meta.NameTitles {
				for _, cand := range titleCandidates(sent, title) {
					if _, ok := known[cand]; ok {
						continue
					}
					for name := range known {
						if strings.HasSuffix(name, title) && rules.Similarity(cand, name) >= 0.5 && rules.Lev(cand, name) <= 1 {
							vs = append(vs, state.Violation{
								Gate: state.GateConsistency, Episode: ep.Ep, Position: pos,
								Expected: name, Actual: cand,
								Severity: state.SeverityError,
								Message:  "疑似姓名漂移（与登记名仅一字之差）",
							})
						}
					}
				}
			}
			// 地名/品牌漂移：X+地名后缀 未登记 → warn（登记面外可能合法提及）。
			for _, suffix := range c.Meta.PlaceSuffixes {
				for _, cand := range titleCandidates(sent, suffix) {
					if _, ok := known[cand]; ok {
						continue
					}
					vs = append(vs, state.Violation{
						Gate: state.GateConsistency, Episode: ep.Ep, Position: pos,
						Expected: "登记在 entities 中的地名/品牌（如 宋驿）", Actual: cand,
						Severity: state.SeverityWarn,
						Message:  "未登记的地名/品牌提及（若为剧本内其他商家请登记或改写）",
					})
				}
			}
		}
	}
	return vs
}

// titleCandidates 抽取 sent 中以 title 结尾、前缀 ≤3 个汉字的候选 token。
func titleCandidates(sent, title string) []string {
	rs := []rune(sent)
	tr := []rune(title)
	var out []string
	for i := 0; i+len(tr) <= len(rs); i++ {
		if string(rs[i:i+len(tr)]) != title {
			continue
		}
		start := i
		for start > 0 && i-start < 3 && rules.IsHan(rs[start-1]) {
			start--
		}
		if start == i { // 称谓前无名字
			continue
		}
		out = append(out, string(rs[start:i+len(tr)]))
	}
	return dedup(out)
}

func dedup(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

// Relationship 关系门（issue #1 §B-2 门 4）：在台账已有相遇记录的两人，
// 禁止"初次相识/自我介绍"叙事。拦：E12 崔白对柳青眉二次自我介绍。
func Relationship(c *canon.Canon, eps []state.Episode, snaps []state.Snapshot) []state.Violation {
	var vs []state.Violation
	markers := []string{"初次", "第一次见", "自我介绍", "他叫", "她叫", "这位是"}
	for i, ep := range eps {
		if i == 0 || i >= len(snaps) {
			continue
		}
		before := snaps[i-1] // 进入本集前的世界状态
		sents := rules.Sentences(ep.Text)
		for si, sent := range sents {
			hit := false
			for _, m := range markers {
				if strings.Contains(sent, m) {
					hit = true
					break
				}
			}
			if !hit {
				continue
			}
			namesInSent := namesIn(c, sent)
			var namesNear map[string]string
			if si > 0 {
				namesNear = namesIn(c, sents[si-1])
			} else {
				namesNear = map[string]string{}
			}
			for xName, xID := range namesInSent {
				for yName, yID := range namesNear {
					if xID == yID {
						continue
					}
					if firstEp, met := before.MetBefore(xID, yID); met {
						vs = append(vs, state.Violation{
							Gate: state.GateRelationship, Episode: ep.Ep,
							Position: fmt.Sprintf("第%d句", si+1),
							Expected: fmt.Sprintf("%s 与 %s 已于 E%d 相遇，不得重走初次叙事", xName, yName, firstEp),
							Actual:   sent,
							Severity: state.SeverityError,
							Message:  "二次相识/自我介绍（E12 崔白类缺陷）",
						})
					}
				}
			}
		}
	}
	return vs
}

// namesIn 返回句子中出现的已登记名称 → entity id。
func namesIn(c *canon.Canon, sent string) map[string]string {
	out := map[string]string{}
	for name, id := range c.KnownNames() {
		if strings.Contains(sent, name) {
			out[name] = id
		}
	}
	return out
}
