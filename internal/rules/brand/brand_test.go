package brand

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

// demo 排期表覆盖 E1–E5；eps 声明与排期一致的主卖点。
func run(t *testing.T, texts map[int]string, declared map[int]string) []state.Violation {
	t.Helper()
	c := demoCanon(t)
	eps := make([]state.Episode, 0, len(texts))
	for ep := 1; ep <= 5; ep++ {
		eps = append(eps, state.Episode{
			Ep: ep, Text: texts[ep],
			Delta: state.Delta{SellingPoint: declared[ep]},
		})
	}
	return Rule().Check(rules.Input{Canon: c, Episodes: eps})
}

func has(vs []state.Violation, sub string) bool {
	for _, v := range vs {
		if strings.Contains(v.Message, sub) || strings.Contains(v.Actual, sub) {
			return true
		}
	}
	return false
}

var allDeclared = map[int]string{
	1: "热水淋浴", 2: "个性化响应", 3: "隐私尊重", 4: "危机关怀", 5: "热水淋浴",
}

// 排期集全部申报 + 材质合法 → 零违规。
func TestClean(t *testing.T) {
	vs := run(t, map[int]string{
		1: "他从怀里摸出一枚木牌搁在柜上。",
		4: "崔白把一枚铜令牌搁在柜上，说要押在柜上。",
	}, allDeclared)
	if len(vs) != 0 {
		t.Fatalf("合法排期与材质应零违规，得到：%s", state.FormatViolations(vs))
	}
}

// 排期集未申报卖点 = 该集白卖，硬失败。
func TestMissingDeclaration(t *testing.T) {
	decl := map[int]string{1: "热水淋浴", 2: "个性化响应", 3: "隐私尊重", 4: "危机关怀"}
	vs := run(t, nil, decl) // E5 排了"热水淋浴/胜负手"却未申报
	found := false
	for _, v := range vs {
		if v.Episode == 5 && strings.Contains(v.Message, "未申报卖点") {
			found = true
		}
	}
	if !found {
		t.Fatalf("E5 未申报必须报错，得到：%s", state.FormatViolations(vs))
	}
}

// 复现 issue #1：令牌材质四套并存——tiers 外材质（玉牌）硬失败。
func TestMaterialDrift(t *testing.T) {
	vs := run(t, map[int]string{1: "他从怀里摸出一块玉牌搁在柜上。"}, allDeclared)
	if !has(vs, "材质漂移") {
		t.Fatalf("tiers 外材质必须报错，得到：%s", state.FormatViolations(vs))
	}
	for _, v := range vs {
		if strings.Contains(v.Message, "材质漂移") && v.Actual != "玉牌" {
			t.Fatalf("应点名玉牌，得到：%s", v)
		}
	}
}

// 隔字组合同样拦：铁令牌（铁不在 tiers 木/铜/银/金 内）。
func TestMaterialDriftSpaced(t *testing.T) {
	vs := run(t, map[int]string{1: "柜上摆着一枚铁令牌。"}, allDeclared)
	if !has(vs, "材质漂移") {
		t.Fatalf("铁令牌必须报错，得到：%s", state.FormatViolations(vs))
	}
}

// 非牌类语境的材质字（黄铜的机关）不误报。
func TestNoFalsePositive(t *testing.T) {
	vs := run(t, map[int]string{1: "她拧动黄铜的机关，白汽轰然而起。"}, allDeclared)
	if has(vs, "材质漂移") {
		t.Fatalf("黄铜机关不应误报，得到：%s", state.FormatViolations(vs))
	}
}
