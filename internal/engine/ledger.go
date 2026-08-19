package engine

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Cloudbird-Software/Script_Writer/internal/canon"
	"github.com/Cloudbird-Software/Script_Writer/internal/rules"
	"github.com/Cloudbird-Software/Script_Writer/internal/state"
)

// Result 是一次全量校验的完整产出。
type Result struct {
	Violations  []state.Violation
	Deliverable Deliverable
	Suggestions []Suggestion
}

// HasError 报告是否存在任一硬失败（CLI 退出码依据）。
func (r *Result) HasError() bool {
	for _, v := range r.Violations {
		if v.Severity == state.SeverityError {
			return true
		}
	}
	return r.Deliverable.Blocked
}

// Run 全量编排：canon 结构校验 → 逐集 apply（state 违规）→ 注册表全部门禁 → 结算。
func Run(c *canon.Canon, eps []state.Episode) (*Result, error) {
	var vs []state.Violation

	// 1. canon 结构问题转为违规。
	for _, p := range c.Validate() {
		vs = append(vs, state.Violation{
			Gate: "canon", Episode: 0, Position: p.Table,
			Expected: "结构合法", Actual: p.String(), Severity: state.SeverityError,
			Message: "canon 表结构问题：" + p.Message,
		})
	}

	// 2. 状态台账（顺序 apply，快照供门禁/结算）。
	e := state.NewEngine(c)
	for _, ep := range eps {
		vs = append(vs, e.ApplyEp(ep)...)
	}
	snaps := e.Snapshots()

	// 3. 注册表驱动的全部门禁。
	in := rules.Input{Canon: c, Episodes: eps, Snapshots: snaps}
	for _, r := range allRules() {
		vs = append(vs, r.Check(in)...)
	}

	// 4. 结算 + 巡检建议。
	res := &Result{Violations: vs}
	res.Deliverable = LedgerClose(c, eps, snaps, vs)
	res.Suggestions = Sweep(c, eps)
	return res, nil
}

// ----------------------------------------------------------------------------
// Ledger Close：交付五件套报表（issue #1 §C Pass 2 + 交付物门）
// ----------------------------------------------------------------------------

// CharacterRow 人物表一行。
type CharacterRow struct {
	ID          string
	Name        string
	AppearedEps []int
	AliasesSeen []string
}

// LoopRow 伏笔台账一行。
type LoopRow struct {
	LoopID   string
	Priority string
	Desc     string
	OpenedEp int
	ClosedEp int
	Status   string // open | closed
}

// CoverageRow 卖点覆盖表。
type CoverageRow struct {
	Point    string
	Eps      []int
	Dims     []string
	Category string
}

// BeatRow 每集 beat + 钩子表一行。
type BeatRow struct {
	Ep           int
	HooksOpened  []string
	HooksClosed  []string
	SellingPoint string
	Emotion      string
}

// Deliverable 是交付五件套的数据源。
type Deliverable struct {
	Characters []CharacterRow
	Loops      []LoopRow
	Coverage   []CoverageRow
	Risks      []state.Violation // 风险清单 = 全部 warn 级违规
	Beats      []BeatRow
	Blocked    bool // 任一 P0 伏笔未回收 → 不许交付
}

// LedgerClose 结算三张表并生成五件套；Blocked 依据最终快照的 P0 未回收。
func LedgerClose(c *canon.Canon, eps []state.Episode, snaps []state.Snapshot, allViolations []state.Violation) Deliverable {
	d := Deliverable{}
	known := c.KnownNames()
	aliasToID := map[string]string{}
	for _, e := range c.Entities {
		for _, a := range e.Aliases {
			aliasToID[a] = e.ID
		}
	}
	// 人物表：出场集扫描。
	for _, e := range c.Entities {
		if e.Type != "character" {
			continue
		}
		row := CharacterRow{ID: e.ID, Name: e.CanonicalName}
		for _, ep := range eps {
			hit := false
			for name, id := range known {
				if id == e.ID && strings.Contains(ep.Text, name) {
					hit = true
					if name != e.CanonicalName {
						row.AliasesSeen = appendUnique(row.AliasesSeen, name)
					}
				}
			}
			if hit {
				row.AppearedEps = append(row.AppearedEps, ep.Ep)
			}
		}
		if len(row.AppearedEps) > 0 {
			d.Characters = append(d.Characters, row)
		}
	}
	// 伏笔台账。
	if len(snaps) > 0 {
		final := snaps[len(snaps)-1]
		for _, lp := range final.Loops {
			status := "closed"
			if lp.Open() {
				status = "open"
				if lp.Priority == "P0" {
					d.Blocked = true
				}
			}
			d.Loops = append(d.Loops, LoopRow{
				LoopID: lp.LoopID, Priority: lp.Priority, Desc: lp.Desc,
				OpenedEp: lp.OpenedEp, ClosedEp: lp.ClosedEp, Status: status,
			})
		}
		sort.Slice(d.Loops, func(i, j int) bool { return d.Loops[i].OpenedEp < d.Loops[j].OpenedEp })
	}
	// 卖点覆盖表（以 canon 排期为准，标注实际申报）。
	byPoint := map[string]*CoverageRow{}
	for _, s := range c.SellingPoints.Schedule {
		row, ok := byPoint[s.Point]
		if !ok {
			row = &CoverageRow{Point: s.Point, Category: s.Category}
			byPoint[s.Point] = row
		}
		row.Eps = append(row.Eps, s.Ep)
		row.Dims = append(row.Dims, s.Dimension)
	}
	for _, row := range byPoint {
		d.Coverage = append(d.Coverage, *row)
	}
	sort.Slice(d.Coverage, func(i, j int) bool { return d.Coverage[i].Point < d.Coverage[j].Point })
	// 每集 beat+钩子表。
	for _, ep := range eps {
		b := BeatRow{Ep: ep.Ep, SellingPoint: ep.Delta.SellingPoint, Emotion: ep.Delta.Emotion}
		for _, h := range ep.Delta.HooksOpened {
			b.HooksOpened = append(b.HooksOpened, h.LoopID)
		}
		b.HooksClosed = append(b.HooksClosed, ep.Delta.HooksClosed...)
		d.Beats = append(d.Beats, b)
	}
	// 风险清单。
	for _, v := range allViolations {
		if v.Severity == state.SeverityWarn {
			d.Risks = append(d.Risks, v)
		}
	}
	return d
}

func appendUnique(list []string, s string) []string {
	for _, x := range list {
		if x == s {
			return list
		}
	}
	return append(list, s)
}

// RenderMarkdown 把五件套渲染为 markdown（报表文件内容）。
func (d Deliverable) RenderMarkdown() string {
	var b strings.Builder
	b.WriteString("# 交付五件套（Ledger Close）\n\n")
	b.WriteString(fmt.Sprintf("交付状态：%s\n\n", ternary(d.Blocked, "**BLOCKED**（存在未回收的 P0 伏笔，不许交付）", "可交付（P0 已清账）")))

	b.WriteString("## 一、人物表\n\n| 实体 | 名称 | 出场集 | 别名命中 |\n|---|---|---|---|\n")
	for _, r := range d.Characters {
		b.WriteString(fmt.Sprintf("| %s | %s | %v | %v |\n", r.ID, r.Name, r.AppearedEps, r.AliasesSeen))
	}
	b.WriteString("\n## 二、伏笔台账\n\n| loop | 优先级 | 埋设 | 回收 | 状态 | 说明 |\n|---|---|---|---|---|---|\n")
	for _, r := range d.Loops {
		closed := "—"
		if r.ClosedEp > 0 {
			closed = fmt.Sprintf("E%d", r.ClosedEp)
		}
		b.WriteString(fmt.Sprintf("| %s | %s | E%d | %s | %s | %s |\n", r.LoopID, r.Priority, r.OpenedEp, closed, r.Status, r.Desc))
	}
	b.WriteString("\n## 三、卖点覆盖表\n\n| 卖点 | 类别 | 排期集 | 维度 |\n|---|---|---|---|\n")
	for _, r := range d.Coverage {
		b.WriteString(fmt.Sprintf("| %s | %s | %v | %v |\n", r.Point, r.Category, r.Eps, r.Dims))
	}
	b.WriteString("\n## 四、风险清单（warn 级）\n\n")
	if len(d.Risks) == 0 {
		b.WriteString("（无）\n")
	}
	for _, v := range d.Risks {
		b.WriteString(fmt.Sprintf("- %s\n", v))
	}
	b.WriteString("\n## 五、每集 beat + 钩子表\n\n| 集 | 开钩 | 收钩 | 卖点 | 情绪 |\n|---|---|---|---|---|\n")
	for _, r := range d.Beats {
		b.WriteString(fmt.Sprintf("| E%d | %v | %v | %s | %s |\n", r.Ep, r.HooksOpened, r.HooksClosed, r.SellingPoint, r.Emotion))
	}
	return b.String()
}

func ternary(cond bool, a, b string) string {
	if cond {
		return a
	}
	return b
}
