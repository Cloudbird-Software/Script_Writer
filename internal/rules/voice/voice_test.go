package voice

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

// 指纹分明：柳青眉短促、宁捕快绵长 → 标准差足够大，零违规。
func TestDistinctVoices(t *testing.T) {
	vs := run(t, "柳青眉说：『水烧好了。』她转身便走。宁捕快说：『这话要说得足够长才能显出捕快的老练与周全来。』他也走了。柳青眉又说：『灯留着。』宁捕快又说：『夜里水路不太平，替我把那盏灯留到天亮再说。』")
	if len(vs) != 0 {
		t.Fatalf("指纹分明应零违规，得到：%s", state.FormatViolations(vs))
	}
}

// 复现 issue #1 软门 13：所有人说话一个味儿 → warn。
func TestSameVoiceWarn(t *testing.T) {
	vs := run(t, "柳青眉说：『今天天气很好啊。』她说完便走。崔白说：『明日大概会下雨。』他也走了。柳青眉又说：『今晚月色不错。』崔白又说：『事情就这么定了。』")
	if len(vs) != 1 || vs[0].Severity != state.SeverityWarn {
		t.Fatalf("一个味儿应为单条 warn，得到：%s", state.FormatViolations(vs))
	}
	if !strings.Contains(vs[0].Actual, "柳青眉") || !strings.Contains(vs[0].Actual, "崔白") {
		t.Fatalf("指纹报告应点名各角色均值，得到：%s", vs[0].Actual)
	}
}

// 样本不足（单人开口/每人仅一句）不判，避免误报。
func TestTooFewSamples(t *testing.T) {
	vs := run(t, "柳青眉说：『水烧好了。』她又说：『灯留着。』")
	if len(vs) != 0 {
		t.Fatalf("单人样本不应判，得到：%s", state.FormatViolations(vs))
	}
}

// 归属歧义（语境含两个具名角色）的台词不入样，不参与指纹。
func TestAmbiguousAttributionSkipped(t *testing.T) {
	vs := run(t, "柳青眉把册子递给崔白：『你自己看吧。』崔白接过去。")
	if len(vs) != 0 {
		t.Fatalf("歧义台词不应入样，得到：%s", state.FormatViolations(vs))
	}
}
