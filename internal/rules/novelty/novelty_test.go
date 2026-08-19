package novelty

import (
	"strings"
	"testing"

	"github.com/Cloudbird-Software/Script_Writer/internal/canon"
	"github.com/Cloudbird-Software/Script_Writer/internal/rules"
	"github.com/Cloudbird-Software/Script_Writer/internal/state"
)

func demoCanon(t *testing.T) *canon.Canon {
	t.Helper()
	c, err := canon.Load("../../canon/testdata/demo")
	if err != nil {
		t.Fatalf("load demo canon: %v", err)
	}
	return c
}

func ep(n int, text string, facts, changes int) state.Episode {
	fs := make([]string, facts)
	for i := range fs {
		fs[i] = "新事实"
	}
	cs := make([]string, changes)
	for i := range cs {
		cs[i] = "状态变化"
	}
	return state.Episode{Ep: n, Text: text, Delta: state.Delta{NewFacts: fs, StateChanges: cs}}
}

func run(t *testing.T, eps ...state.Episode) []state.Violation {
	t.Helper()
	return Rule().Check(rules.Input{Canon: demoCanon(t), Episodes: eps})
}

func has(vs []state.Violation, sub string) bool {
	for _, v := range vs {
		if strings.Contains(v.Message, sub) || strings.Contains(v.Position, sub) {
			return true
		}
	}
	return false
}

// 申报齐全 + 文本全新 → 零违规。
func TestFreshClean(t *testing.T) {
	vs := run(t,
		ep(1, "柳青眉在柜后擦台面，收了铜钱，把牌子推了回去。", 1, 1),
		ep(2, "次日天阴，阿福挑着空担子进店，妇人抱着孩子来避风。", 1, 1),
	)
	if len(vs) != 0 {
		t.Fatalf("全新文本应零违规，得到：%s", state.FormatViolations(vs))
	}
}

// 未申报 new_fact / state_change → 硬失败（复述集判定）。
func TestMissingDeclarations(t *testing.T) {
	vs := run(t, ep(1, "柳青眉擦着柜台。", 0, 1), ep(2, "崔白在楼上抄书。", 1, 0))
	if !has(vs, "delta.new_facts") || !has(vs, "delta.state_changes") {
		t.Fatalf("缺申报必须报错，得到：%s", state.FormatViolations(vs))
	}
}

// 复现 issue #1：E19/E21/E29 三集复述——整段照搬前文必须硬失败。
func TestVerbatimRepeat(t *testing.T) {
	text := "柳青眉提灯引他上楼，推开房门，拧动黄铜的机关，热水从头顶倾下来，白汽轰然而起。"
	vs := run(t, ep(1, text, 1, 1), ep(2, text, 1, 1))
	if !has(vs, "复述") {
		t.Fatalf("整段复述必须报错，得到：%s", state.FormatViolations(vs))
	}
	for _, v := range vs {
		if strings.Contains(v.Message, "复述") && v.Severity != state.SeverityError {
			t.Fatalf("复述应为 error 级：%s", v)
		}
	}
}

// 部分复述低于阈值（默认 60%）不拦——允许承接性短语，拦的是整段套路。
func TestPartialRepeatBelowThreshold(t *testing.T) {
	vs := run(t,
		ep(1, "柳青眉提灯引他上楼，推开房门，拧动黄铜的机关，热水从头顶倾下来，白汽轰然而起。", 1, 1),
		ep(2, "崔白下楼把铜令牌押在柜上，柳青眉收了牌，给他添了一碗热汤，雨点砸在檐上。", 1, 1),
	)
	if has(vs, "复述") {
		t.Fatalf("低于阈值不应报复述，得到：%s", state.FormatViolations(vs))
	}
}

// 比较窗口：只与前 compare_window_eps（默认 3）集比，窗口外不复计。
func TestWindowScope(t *testing.T) {
	text := "柳青眉把父亲留下的底册从樟木箱底翻出来，压在那纸旧契上。"
	// E1 与 E5 同文：窗口只看 E2~E4（互不相同）→ 不报。
	vs := run(t,
		ep(1, text, 1, 1),
		ep(2, "次日天阴，阿福挑着空担子进店。", 1, 1),
		ep(3, "宁捕快立在门外，袖口沾着水汽。", 1, 1),
		ep(4, "雨点砸在檐上，大堂的灯映得满地水光。", 1, 1),
		ep(5, text, 1, 1),
	)
	if has(vs, "复述") {
		t.Fatalf("窗口外重复不应报，得到：%s", state.FormatViolations(vs))
	}
}
