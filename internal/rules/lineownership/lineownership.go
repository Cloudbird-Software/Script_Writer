// Package lineownership 是 M4 台词归属门（issue #1 §B-2 门 6）。
//
// 拦：slogan/口头禅被塞错嘴（E8/E19 云娘、E25 邻店掌柜、E28 宋十两说主角台词）、
// 台词用量申报与正文不符（正文出现但未申报 / 申报了却没写）。
//
// 规则版归属判定：台词所在句恰好含一个具名角色 → 该角色即说话人，
// 与 lines.owner 比对；句中无具名角色或多个角色（歧义）时不判归属（M5 LLM 补代词消解）。
// 单集/全剧次数上限已由 state 台账层（LineUses）校验，本门不再重复。
//
// 深接口纪律：本包只导出 Rule() 一个构造器。
package lineownership

import (
	"fmt"
	"strings"

	"github.com/Cloudbird-Software/Script_Writer/internal/canon"
	"github.com/Cloudbird-Software/Script_Writer/internal/rules"
	"github.com/Cloudbird-Software/Script_Writer/internal/state"
)

// Rule 返回台词归属门禁实例。
func Rule() rules.Rule { return rule{} }

type rule struct{}

// ID 实现 rules.Rule。
func (rule) ID() string { return state.GateLineOwnership }

// Check 实现 rules.Rule。
func (rule) Check(in rules.Input) []state.Violation {
	var vs []state.Violation
	for _, ep := range in.Episodes {
		declared := map[string]int{}
		for _, lu := range ep.Delta.LineUses {
			declared[lu.Line] += lu.Count
		}
		sents := rules.Sentences(ep.Text)
		for _, line := range in.Canon.Lines {
			occurrences, hitSent := 0, -1
			for si, sent := range sents {
				n := countLine(sent, line)
				if n > 0 {
					occurrences += n
					hitSent = si
				}
			}
			// 1. 用量申报与正文一致（正文出现必须申报，申报必须写进正文）。
			if occurrences != declared[line.ID] {
				vs = append(vs, state.Violation{
					Gate: state.GateLineOwnership, Episode: ep.Ep,
					Position: "delta.line_uses." + line.ID,
					Expected: fmt.Sprintf("申报次数 = 正文出现次数（%d）", occurrences),
					Actual:   fmt.Sprint(declared[line.ID]),
					Severity: state.SeverityError,
					Message:  "台词用量申报与正文不符（未申报的使用无法进台账限额）",
				})
			}
			// 2. 归属：台词所在句恰好一个具名角色 → 必须是 owner。
			if hitSent >= 0 {
				if speaker, name := rules.SoleCharacter(in.Canon, sents[hitSent]); speaker != "" && speaker != line.Owner {
					owner := ownerName(in.Canon, line.Owner)
					vs = append(vs, state.Violation{
						Gate: state.GateLineOwnership, Episode: ep.Ep,
						Position: fmt.Sprintf("第%d句", hitSent+1),
						Expected: fmt.Sprintf("仅 %s 可说（owner）", owner),
						Actual:   name,
						Severity: state.SeverityError,
						Message:  "台词塞错嘴（E8/E19/E25/E28 客人抢主角台词类缺陷）",
					})
				}
			}
		}
	}
	return vs
}

// countLine 统计一句中台词资产（原文或变体）出现次数。
func countLine(sent string, line canon.Line) int {
	n := strings.Count(sent, line.Text)
	for _, v := range line.Variants {
		n += strings.Count(sent, v)
	}
	return n
}

func ownerName(c *canon.Canon, id string) string {
	if e, ok := c.EntityByID(id); ok {
		return e.CanonicalName
	}
	return id
}
