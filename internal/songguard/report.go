package songguard

import (
	"encoding/json"
	"fmt"

	"github.com/Cloudbird-Software/Script_Writer/internal/engine"
	"github.com/Cloudbird-Software/Script_Writer/internal/state"
)

// Report 是一次全量校验的对外产出；渲染与统计集中在此，调用方不触内层包。
type Report struct {
	res *engine.Result
}

// Blocked 报告交付是否被拦（P0 伏笔未清账 → 不许交付）。
func (r *Report) Blocked() bool { return r.res.Deliverable.Blocked }

// HasError 报告是否存在任一硬失败（CLI 退出码依据）。
func (r *Report) HasError() bool { return r.res.HasError() }

// AllViolations 返回全部违规（gate/episode/位置/expected/actual/severity/message）。
func (r *Report) AllViolations() []state.Violation { return r.res.Violations }

// Counts 返回 error/warn 级违规数与巡检建议数。
func (r *Report) Counts() (errs, warns, suggestions int) {
	for _, v := range r.res.Violations {
		switch v.Severity {
		case state.SeverityError:
			errs++
		case state.SeverityWarn:
			warns++
		}
	}
	return errs, warns, len(r.res.Suggestions)
}

// RenderDeliverableMarkdown 渲染交付五件套（人物表/伏笔台账/卖点覆盖/风险清单/beat+钩子）。
func (r *Report) RenderDeliverableMarkdown() string { return r.res.Deliverable.RenderMarkdown() }

// RenderSweepMarkdown 渲染一致性巡检建议（只 diff 建议，不重写全文）。
func (r *Report) RenderSweepMarkdown() string {
	return engine.RenderSuggestionsMarkdown(r.res.Suggestions)
}

// ViolationsJSON 返回结构化违规报告（violations.json 文件内容）。
func (r *Report) ViolationsJSON() string { return mustJSONIndent(r.res.Violations) }

// summary 是 stdout 摘要 JSON 的形态（键序与历史版本兼容）。
type summary struct {
	Blocked     bool              `json:"blocked"`
	Errors      int               `json:"errors"`
	Warns       int               `json:"warns"`
	Suggestions int               `json:"suggestions"`
	Violations  []state.Violation `json:"violations"`
}

// SummaryJSON 返回 stdout 摘要 JSON。
func (r *Report) SummaryJSON() string {
	errs, warns, sugg := r.Counts()
	return mustJSONIndent(summary{
		Blocked:     r.Blocked(),
		Errors:      errs,
		Warns:       warns,
		Suggestions: sugg,
		Violations:  r.res.Violations,
	})
}

// LinkageReport 是 ±1 集联动校验的产出。
type LinkageReport struct {
	Ep int
	vs []state.Violation
}

// OK 报告联动承接是否完整（零违规）。
func (l *LinkageReport) OK() bool { return len(l.vs) == 0 }

// WarnOnly 报告是否只有 warn 级提示（如未申报 pickup_keywords）。
func (l *LinkageReport) WarnOnly() bool {
	for _, v := range l.vs {
		if v.Severity == state.SeverityError {
			return false
		}
	}
	return true
}

// Violations 返回联动违规明细。
func (l *LinkageReport) Violations() []state.Violation { return l.vs }

// Render 按结果渲染人读输出（OK 提示或违规清单）。
func (l *LinkageReport) Render() string {
	if l.OK() {
		return fmt.Sprintf("OK: E%d ±1 集联动承接完整（前集钩子已承接、本集钩子已有下文）", l.Ep)
	}
	out := state.FormatViolations(l.vs)
	if l.WarnOnly() {
		out += "OK: 仅有 warn 级提示（如未申报 pickup_keywords），无联动断裂\n"
	}
	return out
}

// HasError 报告是否存在 error 级联动断裂（CLI 退出码依据）。
func (l *LinkageReport) HasError() bool { return !l.WarnOnly() }

func mustJSONIndent(v any) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		// 仅对受控结构序列化，不可达；保守返回空对象。
		return "{}"
	}
	return string(b)
}
