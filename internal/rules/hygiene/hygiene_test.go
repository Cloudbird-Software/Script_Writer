package hygiene

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
	c := demoCanon(t)
	in := rules.Input{
		Canon:     c,
		Episodes:  []state.Episode{{Ep: 1, Text: text}},
		Snapshots: nil,
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

// 复现 issue #1 §二 P2：暖幢栋(E8)/快仗的通道(E18) 乱码片段必须硬失败。
func TestGarbledPatterns(t *testing.T) {
	vs := run(t, "檐下一排暖幢栋的灯次第亮起，照着快仗的通道。")
	if !has(vs, "暖幢栋") || !has(vs, "快仗") {
		t.Fatalf("乱码片段必须报错，得到：%s", state.FormatViolations(vs))
	}
	for _, v := range vs {
		if v.Severity != state.SeverityError {
			t.Fatalf("乱码应为 error 级：%s", v)
		}
	}
}

// 复现 issue #1 §二 P2：愣/抸扇/针砭丝线 错别字必须硬失败并给出修正建议。
func TestTypos(t *testing.T) {
	vs := run(t, "她愣在原地，见他抸扇般摇了摇，低头缝着针砭丝线。")
	if !has(vs, "愣") || !has(vs, "抸扇") || !has(vs, "针砭丝线") {
		t.Fatalf("错别字必须报错，得到：%s", state.FormatViolations(vs))
	}
	found := map[string]string{}
	for _, v := range vs {
		if v.Severity != state.SeverityError {
			t.Fatalf("错别字应为 error 级：%s", v)
		}
		found[v.Actual] = v.Expected
	}
	if found["愣"] != "怔" {
		t.Fatalf("愣 应建议改为 怔，得到：%v", found)
	}
}

// 生僻字表走 warn（TTS 预读风险，不硬拦）。
func TestRareCharsWarn(t *testing.T) {
	c := demoCanon(t)
	c.Config.Hygiene.RareChars = []string{"囧"}
	in := rules.Input{Canon: c, Episodes: []state.Episode{{Ep: 2, Text: "孩子脸上写着一个囧字。"}}}
	vs := Rule().Check(in)
	if len(vs) != 1 || vs[0].Severity != state.SeverityWarn {
		t.Fatalf("生僻字应为单条 warn，得到：%s", state.FormatViolations(vs))
	}
}

// 干净文本零违规；句定位可被 sweep 消费（第N句）。
func TestCleanAndPosition(t *testing.T) {
	vs := run(t, "她怔在门口，半晌才说出话来。")
	if len(vs) != 0 {
		t.Fatalf("干净文本应零违规，得到：%s", state.FormatViolations(vs))
	}
	vs = run(t, "第一句干干净净。第二句里有个愣字。")
	if len(vs) != 1 || vs[0].Position != "第2句" {
		t.Fatalf("应定位到第 2 句，得到：%s", state.FormatViolations(vs))
	}
}
