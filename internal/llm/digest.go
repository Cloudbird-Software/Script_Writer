package llm

import (
	"fmt"
	"strings"

	"github.com/Cloudbird-Software/Script_Writer/internal/canon"
)

// Digest 生成喂给 LLM pass 的 canon 摘要：只保留判一致性所需的硬事实
// （角色注册名/别名/黑名单、道具 tiers 与规则、时间轴、slogan owner），
// 不含 config 阈值——那是规则门禁的参数，与语义巡检无关。
func Digest(c *canon.Canon) string {
	var b strings.Builder
	b.WriteString("【实体注册表】\n")
	for _, e := range c.Entities {
		fmt.Fprintf(&b, "- %s（%s）：canonical=%s", e.ID, e.Type, e.CanonicalName)
		if len(e.Aliases) > 0 {
			fmt.Fprintf(&b, "；别名：%s", strings.Join(e.Aliases, "、"))
		}
		if len(e.ForbiddenNames) > 0 {
			fmt.Fprintf(&b, "；禁用名：%s", strings.Join(e.ForbiddenNames, "、"))
		}
		b.WriteString("\n")
	}
	if len(c.Props) > 0 {
		b.WriteString("【道具规则】\n")
		for _, p := range c.Props {
			fmt.Fprintf(&b, "- %s（%s）：tiers=%v；发放=%s；回收=%s", p.ID, p.Name, p.Tiers, p.IssueRule, p.ReturnRule)
			b.WriteString("\n")
		}
	}
	if len(c.Lines) > 0 {
		b.WriteString("【台词资产】\n")
		for _, l := range c.Lines {
			fmt.Fprintf(&b, "- %s：%q（owner=%s，全剧上限=%d）\n", l.ID, l.Text, l.Owner, l.MaxUsesTotal)
		}
	}
	if len(c.Timeline) > 0 {
		b.WriteString("【时间轴】\n")
		for _, t := range c.Timeline {
			fmt.Fprintf(&b, "- E%d：%s %s %s\n", t.Ep, t.Date, t.Season, t.TimeOfDay)
		}
	}
	return strings.TrimSpace(b.String())
}
