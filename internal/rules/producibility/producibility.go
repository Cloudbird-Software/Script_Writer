// Package producibility 是 M4 可拍性门（issue #1 §B-2 门 12）。
//
// 拦"散文小说"而非"竖屏剧本"的六类问题：
//
//	① 每集具名角色 ≤ max_named_chars_per_ep（拦"4 个绸缎商"式角色资产膨胀）
//	② 每集新角色 ≤ max_new_chars_per_ep（首集免检——开场阵容是 setup）
//	③ 场景 ≤ max_scenes_per_ep（delta.scenes 申报）
//	④ 剧情关键汉字上屏 = 硬失败（onscreen_triggers：刻着/匾额/纸条上……
//	   必须转图案/符号/后期贴字；拦"上屏汉字 7 处"）
//	⑤ 镜头语言禁止出现在散文（camera_terms：镜头一抬/特写……必须进 shot_spec）
//	⑥ 群演场面全剧配额 ≤ max_crowd_scenes（delta.crowd 累计）
//	⑦ 夜景/水汽/儿童/动物打成本标签（warn，进风险清单供制片评估）
//
// 深接口纪律：本包只导出 Rule() 一个构造器。
package producibility

import (
	"fmt"
	"strings"

	"github.com/Cloudbird-Software/Script_Writer/internal/rules"
	"github.com/Cloudbird-Software/Script_Writer/internal/state"
)

// Rule 返回可拍性门禁实例。
func Rule() rules.Rule { return rule{} }

type rule struct{}

// costTags 是固定成本标签词表（issue #1 门 12 ⑥：打标签不硬拦）。
var costTags = []string{
	"夜景", "入夜", "夜深", "夜里", "夜色", // 夜景打灯
	"水汽", "白汽", "雾气", // 水汽/雾效
	"孩子", "孩童", "娃娃", "婴孩", // 儿童演员
	"犬", "马", "驴", "骡", "猫", "鹰", // 动物
}

// ID 实现 rules.Rule。
func (rule) ID() string { return state.GateProducibility }

// Check 实现 rules.Rule。
func (rule) Check(in rules.Input) []state.Violation {
	var vs []state.Violation
	cfg := in.Canon.Config.WithDefaults().Producibility

	// 具名角色索引：entity id → 名称全集（canonical + aliases）。
	type character struct {
		id    string
		names []string
	}
	var chars []character
	for _, e := range in.Canon.Entities {
		if e.Type != "character" {
			continue
		}
		chars = append(chars, character{id: e.ID, names: append([]string{e.CanonicalName}, e.Aliases...)})
	}

	seen := map[string]bool{} // 前文已出场角色
	crowdEps := 0
	for i, ep := range in.Episodes {
		// ① 具名角色 + ② 新角色。
		present := []string{}
		for _, ch := range chars {
			for _, n := range ch.names {
				if n != "" && strings.Contains(ep.Text, n) {
					present = append(present, ch.id)
					break
				}
			}
		}
		if len(present) > cfg.MaxNamedCharsPerEp {
			vs = append(vs, state.Violation{
				Gate: state.GateProducibility, Episode: ep.Ep, Position: "正文具名角色",
				Expected: fmt.Sprintf("每集 ≤%d 个具名角色", cfg.MaxNamedCharsPerEp),
				Actual:   fmt.Sprint(len(present)),
				Severity: state.SeverityError,
				Message:  "角色资产膨胀（竖屏短剧每集扛不住大量具名角色）",
			})
		}
		if i > 0 { // 首集免检：开场阵容是 setup，不是"新角色涌入"
			fresh := 0
			for _, id := range present {
				if !seen[id] {
					fresh++
				}
			}
			if fresh > cfg.MaxNewCharsPerEp {
				vs = append(vs, state.Violation{
					Gate: state.GateProducibility, Episode: ep.Ep, Position: "正文新角色",
					Expected: fmt.Sprintf("每集 ≤%d 个新角色", cfg.MaxNewCharsPerEp),
					Actual:   fmt.Sprint(fresh),
					Severity: state.SeverityError,
					Message:  "新角色涌入过快（观众记不住，拍摄成本失控）",
				})
			}
		}
		for _, id := range present {
			seen[id] = true
		}

		// ③ 场景数。
		if n := len(ep.Delta.Scenes); n > cfg.MaxScenesPerEp {
			vs = append(vs, state.Violation{
				Gate: state.GateProducibility, Episode: ep.Ep, Position: "delta.scenes",
				Expected: fmt.Sprintf("每集 ≤%d 个场景", cfg.MaxScenesPerEp), Actual: fmt.Sprint(n),
				Severity: state.SeverityError, Message: "场景超限（转场成本）",
			})
		}

		// ④ 上屏汉字 + ⑤ 镜头语言 + ⑦ 成本标签：逐句扫描。
		for si, sent := range rules.Sentences(ep.Text) {
			pos := fmt.Sprintf("第%d句", si+1)
			for _, tr := range cfg.OnscreenTriggers {
				if strings.Contains(sent, tr) {
					vs = append(vs, state.Violation{
						Gate: state.GateProducibility, Episode: ep.Ep, Position: pos,
						Expected: "转图案/符号/后期贴字", Actual: tr,
						Severity: state.SeverityError,
						Message:  "剧情关键汉字上屏（竖屏剧道具字无法拍，必须转图案/符号）",
					})
				}
			}
			for _, ct := range cfg.CameraTerms {
				if strings.Contains(sent, ct) {
					vs = append(vs, state.Violation{
						Gate: state.GateProducibility, Episode: ep.Ep, Position: pos,
						Expected: "写入 shot_spec 字段", Actual: ct,
						Severity: state.SeverityError,
						Message:  "镜头语言混入散文（散文是竖屏剧本，不是分镜脚本）",
					})
				}
			}
			for _, tag := range costTags {
				if strings.Contains(sent, tag) {
					vs = append(vs, state.Violation{
						Gate: state.GateProducibility, Episode: ep.Ep, Position: pos,
						Expected: "成本标签（制片评估）", Actual: tag,
						Severity: state.SeverityWarn,
						Message:  fmt.Sprintf("成本标签：%s（夜景/水汽/儿童/动物拍摄成本提示）", tag),
					})
				}
			}
		}

		// ⑥ 群演配额。
		if ep.Delta.Crowd {
			crowdEps++
			if crowdEps > cfg.MaxCrowdScenes {
				vs = append(vs, state.Violation{
					Gate: state.GateProducibility, Episode: ep.Ep, Position: "delta.crowd",
					Expected: fmt.Sprintf("全剧 ≤%d 场群演", cfg.MaxCrowdScenes),
					Actual:   fmt.Sprintf("第 %d 场", crowdEps),
					Severity: state.SeverityError, Message: "群演场次超全剧配额",
				})
			}
		}
	}
	return vs
}
