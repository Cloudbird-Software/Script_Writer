// Package hygiene 是 M4 文本卫生门（issue #1 §B-2 门 2）。
//
// 拦：错别字/异体字（愣→怔、抸扇→折扇、针砭丝线→针线）、乱码/未成词片段
// （暖幢栋、快仗的通道）、生僻字（供 TTS 预读，warn 级）。
// 词表来自 canon config.yaml（缺省走 WithDefaults，默认词表取材 issue #1 已发生缺陷）。
// 失败处理：硬失败，定点重写该句。
//
// 深接口纪律：本包只导出 Rule() 一个构造器，engine 注册表一行接入。
package hygiene

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Cloudbird-Software/Script_Writer/internal/rules"
	"github.com/Cloudbird-Software/Script_Writer/internal/state"
)

// Rule 返回文本卫生门禁实例。
func Rule() rules.Rule { return rule{} }

type rule struct{}

// ID 实现 rules.Rule。
func (rule) ID() string { return state.GateHygiene }

// Check 实现 rules.Rule：逐句扫描三张词表。
func (rule) Check(in rules.Input) []state.Violation {
	var vs []state.Violation
	cfg := in.Canon.Config.WithDefaults().Hygiene
	// typos 的 key 排序，保证违规输出稳定（map 迭代乱序）。
	typos := make([]string, 0, len(cfg.Typos))
	for k := range cfg.Typos {
		typos = append(typos, k)
	}
	sort.Slice(typos, func(i, j int) bool { return len([]rune(typos[i])) > len([]rune(typos[j])) })

	for _, ep := range in.Episodes {
		for si, sent := range rules.Sentences(ep.Text) {
			pos := fmt.Sprintf("第%d句", si+1)
			for _, typo := range typos {
				if strings.Contains(sent, typo) {
					vs = append(vs, state.Violation{
						Gate: state.GateHygiene, Episode: ep.Ep, Position: pos,
						Expected: cfg.Typos[typo], Actual: typo,
						Severity: state.SeverityError,
						Message:  fmt.Sprintf("错别字/异体字（建议改为 %q，TTS 会读错）", cfg.Typos[typo]),
					})
				}
			}
			for _, g := range cfg.GarbledPatterns {
				if strings.Contains(sent, g) {
					vs = append(vs, state.Violation{
						Gate: state.GateHygiene, Episode: ep.Ep, Position: pos,
						Expected: "成词的现代汉语", Actual: g,
						Severity: state.SeverityError,
						Message:  "乱码/未成词片段（整段语义失效风险，无法配音）",
					})
				}
			}
			for _, rc := range cfg.RareChars {
				if strings.ContainsRune(sent, []rune(rc)[0]) {
					vs = append(vs, state.Violation{
						Gate: state.GateHygiene, Episode: ep.Ep, Position: pos,
						Expected: "常用字表", Actual: rc,
						Severity: state.SeverityWarn,
						Message:  "生僻字（TTS 预读风险，请标注读音或改写）",
					})
				}
			}
		}
	}
	return vs
}
