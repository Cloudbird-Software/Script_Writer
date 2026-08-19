package arc

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

type epArc struct {
	ep     int
	level  int
	cost   string
	finale bool
}

func run(t *testing.T, eps ...epArc) []state.Violation {
	t.Helper()
	list := make([]state.Episode, len(eps))
	for i, e := range eps {
		list[i] = state.Episode{
			Ep: e.ep, Finale: e.finale,
			Delta: state.Delta{Arc: &state.ArcAdvance{Level: e.level, Cost: e.cost}},
		}
	}
	return Rule().Check(rules.Input{Canon: demoCanon(t), Episodes: list})
}

func has(vs []state.Violation, sub string) bool {
	for _, v := range vs {
		if strings.Contains(v.Message, sub) || strings.Contains(v.Expected, sub) {
			return true
		}
	}
	return false
}

// 健康弧线：E1 起步 0，E5 升 1（有代价），E10 升 2——零违规。
func TestHealthyArc(t *testing.T) {
	vs := run(t,
		epArc{ep: 1, level: 0},
		epArc{ep: 5, level: 1, cost: "交出地契底册"},
		epArc{ep: 10, level: 2, cost: "当众认下旧账"},
	)
	if len(vs) != 0 {
		t.Fatalf("健康弧线应零违规，得到：%s", state.FormatViolations(vs))
	}
}

// 复现 issue #1：开局即内部人（E1 已在柜台后、兜里已揣令牌）。
func TestStartsInside(t *testing.T) {
	vs := run(t, epArc{ep: 1, level: 3})
	if !has(vs, "开局即内部人") {
		t.Fatalf("开局非零必须报错，得到：%s", state.FormatViolations(vs))
	}
}

// 升级必须填写 cost——没有代价的升级不产生情绪。
func TestUpgradeNeedsCost(t *testing.T) {
	vs := run(t, epArc{ep: 1, level: 0}, epArc{ep: 5, level: 1})
	if !has(vs, "cost") {
		t.Fatalf("升级缺 cost 必须报错，得到：%s", state.FormatViolations(vs))
	}
}

// 复现 issue #1：E7 就当老板——升级过快，到结局已当了 23 集掌柜。
func TestUpgradeTooFast(t *testing.T) {
	vs := run(t, epArc{ep: 1, level: 0}, epArc{ep: 3, level: 1, cost: "有代价"})
	if !has(vs, "升级过快") {
		t.Fatalf("间隔不足必须报错，得到：%s", state.FormatViolations(vs))
	}
	// 第二次升级同样受限：E5 升 1 后 E8 就升 2。
	vs = run(t,
		epArc{ep: 1, level: 0},
		epArc{ep: 5, level: 1, cost: "有代价"},
		epArc{ep: 8, level: 2, cost: "有代价"},
	)
	if !has(vs, "升级过快") {
		t.Fatalf("二次升级过快必须报错，得到：%s", state.FormatViolations(vs))
	}
}

// 越过 max_level（默认 5）。
func TestOverMaxLevel(t *testing.T) {
	vs := run(t, epArc{ep: 1, level: 0}, epArc{ep: 6, level: 6, cost: "有代价"})
	if !has(vs, "≤5") {
		t.Fatalf("越上限必须报错，得到：%s", state.FormatViolations(vs))
	}
}

// 结局集仍是 0 级 = 30 集零成长（warn）。
func TestFinaleZeroGrowth(t *testing.T) {
	vs := run(t, epArc{ep: 1, level: 0}, epArc{ep: 5, level: 0, finale: true})
	if len(vs) != 1 || vs[0].Severity != state.SeverityWarn {
		t.Fatalf("结局零成长应为单条 warn，得到：%s", state.FormatViolations(vs))
	}
}
