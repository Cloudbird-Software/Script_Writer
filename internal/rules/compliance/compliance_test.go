package compliance

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

func run(t *testing.T, text string) []state.Violation {
	t.Helper()
	in := rules.Input{
		Canon:    demoCanon(t),
		Episodes: []state.Episode{{Ep: 1, Text: text}},
	}
	return Rule().Check(in)
}

func has(vs []state.Violation, sub string) bool {
	for _, v := range vs {
		if strings.Contains(v.Message, sub) || strings.Contains(v.Actual, sub) {
			return true
		}
	}
	return false
}

// 复现 issue #1：E14"替你铺条官路"——官商往来/权力寻租，hard 红线。
func TestHardLine(t *testing.T) {
	vs := run(t, "那官差压低声音道：这事好办，回头我替你铺条官路。")
	if !has(vs, "铺条官路") {
		t.Fatalf("官路红线必须报错，得到：%s", state.FormatViolations(vs))
	}
	for _, v := range vs {
		if v.Actual == "铺条官路" && v.Severity != state.SeverityError {
			t.Fatalf("hard 红线应为 error 级：%s", v)
		}
	}
}

// 复现 issue #1：E27 会员体系表述为特权炫耀（"有令牌才算有脸面"）。
func TestPrivilegeHard(t *testing.T) {
	vs := run(t, "掌柜的冷笑：在宋驿，有令牌才算有脸面。")
	if !has(vs, "有令牌才算有脸面") {
		t.Fatalf("特权表述必须报错，得到：%s", state.FormatViolations(vs))
	}
}

// 复现 issue #1：绝对化用语（头一份）进标记清单——warn 级人审，不硬拦。
func TestAbsoluteFlag(t *testing.T) {
	vs := run(t, "跑堂的竖起大拇指：客官这待遇，汴京头一份！")
	found := false
	for _, v := range vs {
		if v.Actual == "头一份" {
			found = true
			if v.Severity != state.SeverityWarn {
				t.Fatalf("flag 应为 warn 级：%s", v)
			}
		}
	}
	if !found {
		t.Fatalf("头一份必须进人审清单，得到：%s", state.FormatViolations(vs))
	}
}

// 干净文本零违规。
func TestClean(t *testing.T) {
	vs := run(t, "柳青眉把热茶递过去，说夜深雨大，安睡无妨。")
	if len(vs) != 0 {
		t.Fatalf("干净文本应零违规，得到：%s", state.FormatViolations(vs))
	}
}
