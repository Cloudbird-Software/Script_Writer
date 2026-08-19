package lineownership

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

func run(t *testing.T, text string, uses ...state.LineUse) []state.Violation {
	t.Helper()
	in := rules.Input{
		Canon:    demoCanon(t),
		Episodes: []state.Episode{{Ep: 1, Text: text, Delta: state.Delta{LineUses: uses}}},
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

// owner 亲口说 + 申报一致 → 零违规。
func TestOwnerSpeaksDeclared(t *testing.T) {
	vs := run(t, "柳青眉把抹布搁下，只说：『门里门外，规矩我给您圆上。』",
		state.LineUse{Line: "SLOGAN_1", Count: 1})
	if len(vs) != 0 {
		t.Fatalf("owner 台词应零违规，得到：%s", state.FormatViolations(vs))
	}
}

// 复现 issue #1 §二 P2：E28 宋十两说主角台词——slogan 被非 owner 说出必须硬失败。
func TestWrongSpeaker(t *testing.T) {
	vs := run(t, "崔白拱手赔笑：『门里门外，规矩我给您圆上。』",
		state.LineUse{Line: "SLOGAN_1", Count: 1})
	if !has(vs, "台词塞错嘴") {
		t.Fatalf("非 owner 说 slogan 必须报错，得到：%s", state.FormatViolations(vs))
	}
	for _, v := range vs {
		if strings.Contains(v.Message, "塞错嘴") {
			if v.Severity != state.SeverityError {
				t.Fatalf("塞错嘴应为 error 级：%s", v)
			}
			if v.Expected != "仅 柳青眉 可说（owner）" {
				t.Fatalf("期望应点名 owner，得到：%s", v.Expected)
			}
		}
	}
}

// 变体同受归属约束（"规矩，我给您圆上" 也是柳青眉的台词）。
func TestVariantWrongSpeaker(t *testing.T) {
	vs := run(t, "宁捕快咧嘴一笑：『规矩，我给您圆上。』",
		state.LineUse{Line: "SLOGAN_1", Count: 1})
	if !has(vs, "台词塞错嘴") {
		t.Fatalf("变体被抢也必须报错，得到：%s", state.FormatViolations(vs))
	}
}

// 复现 issue #1 §B-2 门 6：正文出现但未申报——未申报的使用无法进台账限额。
func TestUndeclaredUse(t *testing.T) {
	vs := run(t, "柳青眉把抹布搁下，只说：『门里门外，规矩我给您圆上。』")
	if !has(vs, "申报与正文不符") {
		t.Fatalf("正文出现未申报必须报错，得到：%s", state.FormatViolations(vs))
	}
}

// 申报了却没写进正文 = 虚报用量，同样拦。
func TestDeclaredButAbsent(t *testing.T) {
	vs := run(t, "柳青眉低头擦着柜台，没有说话。",
		state.LineUse{Line: "SLOGAN_1", Count: 1})
	if !has(vs, "申报与正文不符") {
		t.Fatalf("申报与正文不符必须报错，得到：%s", state.FormatViolations(vs))
	}
}

// 句中出现两个具名角色 = 归属歧义，规则版不判（留给 M5 LLM 代词消解）。
func TestAmbiguousNotJudged(t *testing.T) {
	vs := run(t, "崔白望着柳青眉，听她说：『门里门外，规矩我给您圆上。』",
		state.LineUse{Line: "SLOGAN_1", Count: 1})
	if len(vs) != 0 {
		t.Fatalf("歧义句不应判归属，得到：%s", state.FormatViolations(vs))
	}
}
