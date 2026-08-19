// Package emotion 是 M4 情绪曲线门（issue #1 §B-2 软门 14）。
//
// 拦：30 集一个套路——每集必须申报情绪类型且在类型表内，
// 连续 max_streak 集同类型即硬失败（这一道门单独就能打散"同一惊叹模板连播"）。
//
// 类型表与连续上限来自 canon config（ADR-0002），缺省走 WithDefaults()
// （惊叹/共情/打脸/温暖/悬疑/危机，连续 3 集同类型 fail）。
//
// 深接口纪律：本包只导出 Rule() 一个构造器。
package emotion

import (
	"fmt"

	"github.com/Cloudbird-Software/Script_Writer/internal/rules"
	"github.com/Cloudbird-Software/Script_Writer/internal/state"
)

// Rule 返回情绪曲线门禁实例。
func Rule() rules.Rule { return rule{} }

type rule struct{}

// ID 实现 rules.Rule。
func (rule) ID() string { return state.GateEmotion }

// Check 实现 rules.Rule。
func (rule) Check(in rules.Input) []state.Violation {
	var vs []state.Violation
	cfg := in.Canon.Config.WithDefaults().Emotion
	types := map[string]bool{}
	for _, t := range cfg.Types {
		types[t] = true
	}

	streak := 0 // 连续同类型计数
	prev := ""
	for _, ep := range in.Episodes {
		e := ep.Delta.Emotion
		if e == "" {
			vs = append(vs, state.Violation{
				Gate: state.GateEmotion, Episode: ep.Ep, Position: "delta.emotion",
				Expected: "申报本集情绪类型", Actual: "空",
				Severity: state.SeverityError,
				Message:  "未申报情绪类型（情绪曲线不可校验，30集一个套路的根源）",
			})
			streak, prev = 0, ""
			continue
		}
		if !types[e] {
			vs = append(vs, state.Violation{
				Gate: state.GateEmotion, Episode: ep.Ep, Position: "delta.emotion",
				Expected: fmt.Sprintf("类型表内之一（%v）", cfg.Types), Actual: e,
				Severity: state.SeverityError,
				Message:  "情绪类型不在类型表内（类型表是资产，进 canon config）",
			})
			streak, prev = 0, ""
			continue
		}
		if e == prev {
			streak++
		} else {
			streak = 1
		}
		if streak == cfg.MaxStreak {
			vs = append(vs, state.Violation{
				Gate: state.GateEmotion, Episode: ep.Ep, Position: "delta.emotion",
				Expected: fmt.Sprintf("连续同类型 <%d 集（%s 已连 %d 集）", cfg.MaxStreak, e, streak),
				Actual:   e, Severity: state.SeverityError,
				Message: "情绪曲线躺平（连续同类型，观众在此流失；issue #1 软门 14）",
			})
		}
		prev = e
	}
	return vs
}
