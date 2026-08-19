package producibility

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

func run(t *testing.T, eps ...state.Episode) []state.Violation {
	t.Helper()
	return Rule().Check(rules.Input{Canon: demoCanon(t), Episodes: eps})
}

func has(vs []state.Violation, sub string) bool {
	for _, v := range vs {
		if strings.Contains(v.Message, sub) || strings.Contains(v.Actual, sub) {
			return true
		}
	}
	return false
}

// 基线干净：角色/场景/上屏/镜头全合规 → 零违规。
func TestCleanBaseline(t *testing.T) {
	vs := run(t,
		state.Episode{Ep: 1, Text: "柳青眉把崔白安顿进客房。"},
		state.Episode{Ep: 2, Text: "柳青眉低头擦着柜台。"},
	)
	if len(vs) != 0 {
		t.Fatalf("基线应零违规，得到：%s", state.FormatViolations(vs))
	}
}

// ① 复现 issue #1"4 个绸缎商"：具名角色超限（阈值收紧到 1 验证判定）。
func TestTooManyNamedChars(t *testing.T) {
	c := demoCanon(t)
	c.Config.Producibility.MaxNamedCharsPerEp = 1
	in := rules.Input{Canon: c, Episodes: []state.Episode{{Ep: 1, Text: "柳青眉与崔白隔柜相望。"}}}
	vs := Rule().Check(in)
	if !has(vs, "具名角色") {
		t.Fatalf("具名角色超限必须报错，得到：%s", state.FormatViolations(vs))
	}
}

// ② 新角色涌入：首集免检，次集两个新角色即拦。
func TestNewCharsRush(t *testing.T) {
	vs := run(t,
		state.Episode{Ep: 1, Text: "柳青眉和崔白说话。"},
		state.Episode{Ep: 2, Text: "宁捕快带着阿福进了大堂。"},
	)
	if !has(vs, "新角色") {
		t.Fatalf("新角色超限必须报错，得到：%s", state.FormatViolations(vs))
	}
}

// ③ 场景超限。
func TestTooManyScenes(t *testing.T) {
	vs := run(t, state.Episode{
		Ep: 1, Text: "柳青眉在大堂里擦柜台。",
		Delta: state.Delta{Scenes: []string{"大堂", "厨房", "后院", "街口"}},
	})
	if !has(vs, "场景超限") {
		t.Fatalf("场景超限必须报错，得到：%s", state.FormatViolations(vs))
	}
}

// ④ 复现 issue #1"上屏汉字 7 处"：剧情关键汉字上屏 = 硬失败。
func TestOnscreenText(t *testing.T) {
	vs := run(t, state.Episode{Ep: 1, Text: "门口的匾额上写着宋驿两个字。"})
	if !has(vs, "上屏") {
		t.Fatalf("上屏汉字必须报错，得到：%s", state.FormatViolations(vs))
	}
	for _, v := range vs {
		if strings.Contains(v.Message, "上屏") && v.Severity != state.SeverityError {
			t.Fatalf("上屏汉字应为 error 级：%s", v)
		}
	}
}

// ⑤ 复现 issue #1"镜头一抬"：镜头语言混入散文。
func TestCameraTerms(t *testing.T) {
	vs := run(t, state.Episode{Ep: 1, Text: "镜头一抬，柳青眉的脸上有了笑意。"})
	if !has(vs, "镜头语言") {
		t.Fatalf("镜头语言必须报错，得到：%s", state.FormatViolations(vs))
	}
}

// ⑥ 群演配额：全剧 ≤2 场，第 3 场报错。
func TestCrowdQuota(t *testing.T) {
	vs := run(t,
		state.Episode{Ep: 1, Text: "柳青眉擦柜台。", Delta: state.Delta{Crowd: true}},
		state.Episode{Ep: 2, Text: "柳青眉擦柜台。", Delta: state.Delta{Crowd: true}},
		state.Episode{Ep: 3, Text: "柳青眉擦柜台。", Delta: state.Delta{Crowd: true}},
	)
	if !has(vs, "群演") {
		t.Fatalf("群演超配额必须报错，得到：%s", state.FormatViolations(vs))
	}
}

// ⑦ 成本标签：水汽/儿童/动物 → warn 进风险清单，不硬拦。
func TestCostTagsWarn(t *testing.T) {
	vs := run(t, state.Episode{Ep: 1, Text: "檐下白汽升起来，孩子看得出了神。"})
	if len(vs) != 2 {
		t.Fatalf("两处成本标签应各报一条 warn，得到：%s", state.FormatViolations(vs))
	}
	for _, v := range vs {
		if v.Severity != state.SeverityWarn {
			t.Fatalf("成本标签应为 warn 级：%s", v)
		}
	}
}
