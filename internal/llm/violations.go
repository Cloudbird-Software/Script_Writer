package llm

import (
	"fmt"
	"strings"

	"github.com/Cloudbird-Software/Script_Writer/internal/state"
)

// LLM 旁路产出的违规门名（一律 warn 级，进风险清单与人审，不阻断交付）。
const (
	GateSweep  = "llm-sweep"
	GateReader = "llm-reader"
)

// Violations 把 pass 报告转为 warn 级违规（进风险清单）。
// LLM 结论是建议级：模型可能错，必须经人审，不能进入阻断链路。
func (r *Report) Violations() []state.Violation {
	var vs []state.Violation
	lastEp := 0
	switch r.Pass {
	case PassSweep:
		for _, f := range r.Findings {
			lastEp = f.Episode
			vs = append(vs, state.Violation{
				Gate: GateSweep, Episode: f.Episode,
				Position: fmt.Sprintf("%s（confidence=%s）", f.Position, f.Confidence),
				Expected: "与 canon / 前文一致",
				Actual:   f.Issue,
				Severity: state.SeverityWarn,
				Message:  "LLM 巡检建议：" + f.Suggestion,
			})
		}
	case PassReader:
		for _, h := range r.Hooks {
			lastEp = h.Episode
			if h.Strength == "weak" {
				vs = append(vs, state.Violation{
					Gate: GateReader, Episode: h.Episode, Position: "结尾钩子",
					Expected: "strong | medium", Actual: h.Strength,
					Severity: state.SeverityWarn,
					Message:  "观众模拟：弱钩子（" + h.Reason + "），追更动力流失点",
				})
			}
		}
		if r.DropOffPrediction != "" {
			vs = append(vs, state.Violation{
				Gate: GateReader, Episode: lastEp, Position: "弃剧点预测",
				Expected: "无重复套路耗尽", Actual: r.DropOffPrediction,
				Severity: state.SeverityWarn,
				Message:  "观众模拟：弃剧风险（建议级，人审确认）",
			})
		}
		if r.TokenRuleConsistent != nil && !*r.TokenRuleConsistent {
			vs = append(vs, state.Violation{
				Gate: GateReader, Episode: lastEp, Position: "令牌复述测试",
				Expected: "正文复述出唯一无矛盾的规则", Actual: r.TokenRuleRestate,
				Severity: state.SeverityWarn,
				Message:  "观众模拟：会员体系规则矛盾（复述测试失败，建议级）",
			})
		}
	}
	return vs
}

// Unavailable 是 sidecar 不可用时的降级违规（warn：旁路失败不影响主流程，
// 但必须在风险清单里可见）。
func Unavailable(pass string, err error) state.Violation {
	return state.Violation{
		Gate: GateSweep, Episode: 0, Position: "sidecar",
		Expected: "LLM 旁路可达", Actual: err.Error(),
		Severity: state.SeverityWarn,
		Message:  fmt.Sprintf("LLM pass %q 不可用（降级，规则门禁不受影响）：%s", pass, strings.TrimSpace(err.Error())),
	}
}
