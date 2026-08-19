package emotion

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

func run(t *testing.T, emotions ...string) []state.Violation {
	t.Helper()
	eps := make([]state.Episode, len(emotions))
	for i, e := range emotions {
		eps[i] = state.Episode{Ep: i + 1, Delta: state.Delta{Emotion: e}}
	}
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

// 交错情绪曲线零违规。
func TestAlternatingClean(t *testing.T) {
	vs := run(t, "惊叹", "温暖", "惊叹", "温暖")
	if len(vs) != 0 {
		t.Fatalf("交错曲线应零违规，得到：%s", state.FormatViolations(vs))
	}
}

// 连续 2 集（< max_streak=3）同类型不拦。
func TestStreakBelowLimit(t *testing.T) {
	vs := run(t, "温暖", "温暖", "惊叹")
	if len(vs) != 0 {
		t.Fatalf("连续 2 集不应拦，得到：%s", state.FormatViolations(vs))
	}
}

// 复现 issue #1 软门 14：连续 3 集同类型即 fail——这一道门单独打散"30集一个套路"。
func TestStreakFail(t *testing.T) {
	vs := run(t, "惊叹", "温暖", "温暖", "温暖")
	if len(vs) != 1 {
		t.Fatalf("连续 3 集同类型应恰好报 1 次（首次达标集），得到：%s", state.FormatViolations(vs))
	}
	if vs[0].Episode != 4 || vs[0].Severity != state.SeverityError {
		t.Fatalf("应在第 4 集报 error，得到：%s", vs[0])
	}
	// 更长连续也只报首次达标（已报过的同一躺平段不重复计）。
	vs = run(t, "温暖", "温暖", "温暖", "温暖")
	if len(vs) != 1 {
		t.Fatalf("4 连温暖应仍只报 1 次，得到：%s", state.FormatViolations(vs))
	}
}

// 未申报情绪类型 = 曲线不可校验，硬失败。
func TestMissingEmotion(t *testing.T) {
	vs := run(t, "惊叹", "")
	if !has(vs, "未申报情绪类型") {
		t.Fatalf("未申报必须报错，得到：%s", state.FormatViolations(vs))
	}
}

// 类型表外申报（类型表是资产，进 canon config）。
func TestUnknownType(t *testing.T) {
	vs := run(t, "惊叹", "爽")
	if !has(vs, "类型表内") {
		t.Fatalf("表外类型必须报错，得到：%s", state.FormatViolations(vs))
	}
	// 表外类型不参与 streak（错误已经报了，不叠加躺平告警）。
	if len(vs) != 1 {
		t.Fatalf("应恰好报 1 次，得到：%s", state.FormatViolations(vs))
	}
}
