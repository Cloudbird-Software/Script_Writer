package passes

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Cloudbird-Software/Script_Writer/internal/canon"
	"github.com/Cloudbird-Software/Script_Writer/internal/gates"
	"github.com/Cloudbird-Software/Script_Writer/internal/state"
)

// Suggestion 是一致性巡检（规则版）的 diff 建议——只建议，绝不重写全文
// （issue #1 §C Pass 1：重写会引入新漂移）。
type Suggestion struct {
	Episode  int    `json:"episode"`
	Sentence int    `json:"sentence"` // 句序（1 起）
	Excerpt  string `json:"excerpt"`
	Problem  string `json:"problem"`
	Suggest  string `json:"suggest"`
}

func (s Suggestion) String() string {
	return fmt.Sprintf("[E%d, 第%d句] %q → %s（%s）", s.Episode, s.Sentence, s.Excerpt, s.Suggest, s.Problem)
}

// Sweep 基于 M2 格式门与一致性门的产出，生成逐句 diff 建议。
func Sweep(c *canon.Canon, eps []state.Episode) []Suggestion {
	var out []Suggestion
	vs := append(
		gates.Format(c, eps, nil),
		gates.Consistency(c, eps, nil)...,
	)
	byEp := map[int][]state.Episode{}
	for _, ep := range eps {
		byEp[ep.Ep] = append(byEp[ep.Ep], ep)
	}
	for _, v := range vs {
		epList, ok := byEp[v.Episode]
		if !ok || len(epList) == 0 {
			continue
		}
		sentNo := parseSentenceNo(v.Position)
		if sentNo < 1 {
			continue
		}
		sents := gates.Sentences(epList[0].Text)
		if sentNo > len(sents) {
			continue
		}
		out = append(out, Suggestion{
			Episode:  v.Episode,
			Sentence: sentNo,
			Excerpt:  sents[sentNo-1],
			Problem:  v.Message,
			Suggest:  suggestFor(v),
		})
	}
	return out
}

// parseSentenceNo 从 "第N句" / "第N句 xxx" 提取 N；无法解析返回 0。
func parseSentenceNo(pos string) int {
	pos = strings.SplitN(pos, " ", 2)[0]
	if !strings.HasPrefix(pos, "第") || !strings.HasSuffix(pos, "句") {
		return 0
	}
	n, err := strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(pos, "第"), "句"))
	if err != nil {
		return 0
	}
	return n
}

func suggestFor(v state.Violation) string {
	switch {
	case strings.Contains(v.Message, "黑名单名称"), strings.Contains(v.Message, "疑似姓名漂移"):
		return "改为 " + v.Expected
	case strings.Contains(v.Message, "混排名"):
		return "改为登记中文名（实体登记名/别名）"
	case strings.Contains(v.Message, "全半角标点混用"):
		return "统一为全角标点"
	case strings.Contains(v.Message, "markdown"):
		return "删除 markdown 标记"
	case strings.Contains(v.Message, "未登记的地名"):
		return "登记进 entities 或改写为已登记地名"
	case strings.Contains(v.Message, "字数越界"):
		return "扩写/精简至目标区间"
	default:
		return "按 " + v.Expected + " 修正"
	}
}

// RenderSuggestionsMarkdown 渲染巡检建议清单（人工确认后逐条 apply，issue #1 §C Pass 1）。
func RenderSuggestionsMarkdown(ss []Suggestion) string {
	var b strings.Builder
	b.WriteString("# 一致性巡检建议（Consistency Sweep · 规则版）\n\n")
	b.WriteString("> 只输出 diff 建议，不重写全文；人工确认后逐条 apply。\n\n")
	if len(ss) == 0 {
		b.WriteString("（无建议）\n")
	}
	for _, s := range ss {
		b.WriteString(fmt.Sprintf("- %s\n", s))
	}
	return b.String()
}
