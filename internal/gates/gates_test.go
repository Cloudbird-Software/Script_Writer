package gates

import (
	"strings"
	"testing"

	rapid "pgregory.net/rapid"

	"github.com/Cloudbird-Software/Script_Writer/internal/canon"
	"github.com/Cloudbird-Software/Script_Writer/internal/state"
)

func demoCanon(t *testing.T) *canon.Canon {
	t.Helper()
	c, err := canon.Load("../canon/testdata/demo")
	if err != nil {
		t.Fatalf("load demo canon: %v", err)
	}
	return c
}

// run 对 eps 依次 apply（收集 state 违规不在本包测试范围）并返回快照。
func run(t *testing.T, c *canon.Canon, eps []state.Episode) []state.Snapshot {
	t.Helper()
	e := state.NewEngine(c)
	for _, ep := range eps {
		e.ApplyEp(ep)
	}
	return e.Snapshots()
}

func hasGateMsg(vs []state.Violation, gate, sub string) bool {
	for _, v := range vs {
		if v.Gate == gate && strings.Contains(v.Message, sub) {
			return true
		}
	}
	return false
}

// ---- 格式门 ----

// 复现 P2：E21≈270 字 vs 目标 500（±10%）。
func TestFormatCharCountReproE21(t *testing.T) {
	c := demoCanon(t)
	short := strings.Repeat("他走进宋驿。", 50) // 250 字
	vs := Format(c, []state.Episode{{Ep: 21, Text: short, TargetChars: 500}}, nil)
	if !hasGateMsg(vs, state.GateFormat, "字数越界") {
		t.Fatalf("E21 字数缺陷未拦截：%s", state.FormatViolations(vs))
	}
	// 合格长度不报（83×6=498 字 ∈ [450,550]）。
	ok := strings.Repeat("他走进宋驿。", 83)
	vs = Format(c, []state.Episode{{Ep: 1, Text: ok, TargetChars: 500}}, nil)
	if hasGateMsg(vs, state.GateFormat, "字数越界") {
		t.Fatalf("合格字数被误报：%s", state.FormatViolations(vs))
	}
}

// 复现 P2：E3/E17/E21/E24 全半角标点混用。
func TestFormatPunctMix(t *testing.T) {
	c := demoCanon(t)
	text := "他走进宋驿,看了看大堂：灯亮着,水是热的。"
	vs := Format(c, []state.Episode{{Ep: 3, Text: text}}, nil)
	if !hasGateMsg(vs, state.GateFormat, "全半角标点混用") {
		t.Fatalf("标点混用未拦截：%s", state.FormatViolations(vs))
	}
}

// 复现 P0-#2：E9"A福"。
func TestFormatMixedScriptNameReproE9(t *testing.T) {
	c := demoCanon(t)
	vs := Format(c, []state.Episode{{Ep: 9, Text: "A福挑着货担进了门，喊了声掌柜的。"}}, nil)
	if !hasGateMsg(vs, state.GateFormat, "A福") {
		t.Fatalf("混排名未拦截：%s", state.FormatViolations(vs))
	}
	// 多字母拉丁词（LED）不误报。
	vs = Format(c, []state.Episode{{Ep: 2, Text: "他打开LED化妆镜，又去调咖啡机。"}}, nil)
	if hasGateMsg(vs, state.GateFormat, "混排名") {
		t.Fatalf("LED 被误报：%s", state.FormatViolations(vs))
	}
}

// 复现 P2：markdown 残留。
func TestFormatMarkdownResidue(t *testing.T) {
	c := demoCanon(t)
	vs := Format(c, []state.Episode{{Ep: 5, Text: "**第一幕**\n他推门进来。"}}, nil)
	if !hasGateMsg(vs, state.GateFormat, "markdown") {
		t.Fatalf("markdown 残留未拦截：%s", state.FormatViolations(vs))
	}
}

// ---- 一致性门 ----

// 复现 P0-#2：E5"渔捕快"漂移（宁捕快黑名单）。
func TestConsistencyForbiddenNameReproE5(t *testing.T) {
	c := demoCanon(t)
	vs := Consistency(c, []state.Episode{{Ep: 5, Text: "渔捕快把船缆系在桩上，抬头看了看宋驿的门牌。"}}, nil)
	if !hasGateMsg(vs, state.GateConsistency, "黑名单名称") {
		t.Fatalf("渔捕快未拦截：%s", state.FormatViolations(vs))
	}
}

// 漂移检测：未登记的"严捕快"与"宁捕快"一字之差 → 疑似漂移。
func TestConsistencyNameDrift(t *testing.T) {
	c := demoCanon(t)
	vs := Consistency(c, []state.Episode{{Ep: 8, Text: "严捕快拱了拱手，径直往里走。"}}, nil)
	if !hasGateMsg(vs, state.GateConsistency, "疑似姓名漂移") {
		t.Fatalf("姓名漂移未拦截：%s", state.FormatViolations(vs))
	}
}

// 复现 P0-#1：E3"靖康驿站"招牌不一致。
func TestConsistencySignboardReproE3(t *testing.T) {
	c := demoCanon(t)
	vs := Consistency(c, []state.Episode{{Ep: 3, Text: "对面靖康驿站的伙计探出头来看热闹。"}}, nil)
	if !hasGateMsg(vs, state.GateConsistency, "黑名单名称") {
		t.Fatalf("靖康驿站未拦截：%s", state.FormatViolations(vs))
	}
}

// 白名单名不误报。
func TestConsistencyNoFalsePositive(t *testing.T) {
	c := demoCanon(t)
	vs := Consistency(c, []state.Episode{{Ep: 1, Text: "宁捕快进了宋驿，柳青眉正在擦柜台。"}}, nil)
	for _, v := range vs {
		if v.Severity == state.SeverityError {
			t.Fatalf("白名单名被误报：%s", v)
		}
	}
}

// ---- 关系门 ----

// 复现 P0-#3：E12 崔白对已相识的柳青眉重新自我介绍。
func TestRelationshipReintroReproE12(t *testing.T) {
	c := demoCanon(t)
	eps := []state.Episode{
		{Ep: 1, Text: "柳青眉立在柜台后。崔白推门进来，讨了碗热水。",
			Delta: state.Delta{Meetings: []state.Meeting{{A: "CUI_BAI", B: "LIU_QINGMEI"}}}},
		{Ep: 12, Text: "柳青眉抬头。一个书生立在柜台前，他叫崔白。"},
	}
	snaps := run(t, c, eps)
	vs := Relationship(c, eps, snaps)
	if !hasGateMsg(vs, state.GateRelationship, "二次相识") {
		t.Fatalf("E12 二次相识未拦截：%s", state.FormatViolations(vs))
	}
	// 未相遇过的两人初次见面不报。
	eps2 := []state.Episode{
		{Ep: 1, Text: "宋驿开张，柳青眉擦拭柜台。"},
		{Ep: 12, Text: "柳青眉抬头。一个书生立在柜台前，他叫崔白。"},
	}
	vs = Relationship(c, eps2, run(t, c, eps2))
	if hasGateMsg(vs, state.GateRelationship, "二次相识") {
		t.Fatalf("正常初见被误报：%s", state.FormatViolations(vs))
	}
}

// ---- 引文接地门 ----

// 复现 P0-#4：E30"过客有期"——全剧无人说过（E23 是"后会有期"，相似度 0.5 <0.8）。
func TestQuoteGroundingReproE30(t *testing.T) {
	c := demoCanon(t)
	eps := []state.Episode{
		{Ep: 23, Text: "崔白拱手道：“后会有期。”说完转身下楼。"},
		{Ep: 30, Text: "柳青眉想起那句『过客有期』，灯花晃了一下。"},
	}
	vs := QuoteGrounding(c, eps, nil)
	if !hasGateMsg(vs, state.GateQuoteGrounding, "引文无出处") {
		t.Fatalf("过客有期未拦截：%s", state.FormatViolations(vs))
	}
	// 逐字出现过的引文接地通过。
	eps2 := []state.Episode{
		{Ep: 23, Text: "崔白拱手道：“后会有期。”说完转身下楼。"},
		{Ep: 30, Text: "柳青眉想起那句『后会有期』，灯花晃了一下。"},
	}
	vs = QuoteGrounding(c, eps2, nil)
	if hasGateMsg(vs, state.GateQuoteGrounding, "引文无出处") {
		t.Fatalf("接地引文被误报：%s", state.FormatViolations(vs))
	}
}

// ---- 钩子/回收门 ----

// 复现 P0-#6：E14→E15 钩子断裂（hook 承接词未出现在下一集开头 300 字）。
func TestHookPayoffAdjacencyReproE14E15(t *testing.T) {
	c := demoCanon(t)
	eps := []state.Episode{
		{Ep: 14, Text: strings.Repeat("宋十两喝多了。", 40),
			Delta: state.Delta{HooksOpened: []state.Hook{
				{LoopID: "L_GUANCHAI", Priority: "P0", PickupKeywords: []string{"官差"}},
			}}},
		{Ep: 15, Text: "账房翻出炭册，一笔一笔对。柳青眉立在旁边。" + strings.Repeat("炭是私屯的。", 40),
			Delta: state.Delta{HooksOpened: []state.Hook{{LoopID: "L_TAN", Priority: "P1", PickupKeywords: []string{"炭册"}}}}},
	}
	vs := HookPayoff(c, eps, run(t, c, eps))
	if !hasGateMsg(vs, state.GateHookPayoff, "相邻集钩子断裂") {
		t.Fatalf("E14→E15 断裂未拦截：%s", state.FormatViolations(vs))
	}
}

// 复现 P2：九集"暖收"无钩子。
func TestHookPayoffNoHookWarmEnding(t *testing.T) {
	c := demoCanon(t)
	eps := []state.Episode{
		{Ep: 4, Text: "热水从花洒落下来，客人舒了口气。"},
		{Ep: 5, Text: "灯亮了一夜。", Delta: state.Delta{HooksOpened: []state.Hook{{LoopID: "L1", PickupKeywords: []string{"灯"}}}}},
	}
	vs := HookPayoff(c, eps, run(t, c, eps))
	if !hasGateMsg(vs, state.GateHookPayoff, "无钩子暖收") {
		t.Fatalf("暖收未拦截：%s", state.FormatViolations(vs))
	}
}

// 悬置时限：>6 告警、>10 硬失败；P0 结局未清账。
func TestHookPayoffAgingAndP0Ledger(t *testing.T) {
	c := demoCanon(t)
	var eps []state.Episode
	for i := 1; i <= 12; i++ {
		d := state.Delta{}
		if i == 1 {
			// 一个永不回收的 P0 钩子 + 一个被承接的普通钩子。
			d.HooksOpened = []state.Hook{
				{LoopID: "L_P0", Priority: "P0", PickupKeywords: []string{"木牌"}},
				{LoopID: "L_N", Priority: "P1", PickupKeywords: []string{"木牌"}},
			}
		}
		eps = append(eps, state.Episode{Ep: i,
			Text:   "木牌在袖中发烫。" + strings.Repeat("柜台擦得发亮。", 30),
			Delta:  d,
			Finale: i == 12,
		})
	}
	vs := HookPayoff(c, eps, run(t, c, eps))
	if !hasGateMsg(vs, state.GateHookPayoff, "超期未回收") {
		t.Fatalf("悬置超期未拦截：%s", state.FormatViolations(vs))
	}
	if !hasGateMsg(vs, state.GateHookPayoff, "P0 伏笔未清账") {
		t.Fatalf("P0 未清账未拦截：%s", state.FormatViolations(vs))
	}
}

// ---- PBT ----

// PBT：纯中文标点文本永不报"混用"；注入任一半角标点必报。
func TestPropertyPunctMixDetection(t *testing.T) {
	fullRunes := []rune(fullWidthPunct)
	halfRunes := []rune(halfWidthPunct)
	rapid.Check(t, func(t *rapid.T) {
		body := rapid.StringMatching(`[一-龥]{1,12}`).Draw(t, "body")
		full := rapid.SampledFrom(fullRunes).Draw(t, "full")
		text := body + string(full)
		if rapid.Bool().Draw(t, "inject") {
			half := rapid.SampledFrom(halfRunes).Draw(t, "half")
			text += string(half)
			if _, _, mixed := PunctMix(text); !mixed {
				t.Fatalf("注入半角 %q 后未判混用：%q", string(half), text)
			}
		} else if _, _, mixed := PunctMix(text); mixed {
			t.Fatalf("纯全角被判混用：%q", text)
		}
	})
}

// PBT：引文只要逐字出现在任意前集，grounded 恒真。
func TestPropertyExactQuoteAlwaysGrounded(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		prior := strings.Join(rapid.SliceOf(rapid.StringMatching(`[一-龥]{2,8}`)).Draw(t, "prior"), "。")
		q := rapid.StringMatching(`[一-龥]{2,6}`).Draw(t, "q")
		corpus := normalize(prior + "。" + q)
		if !grounded(q, corpus) {
			t.Fatalf("逐字出现的引文未接地：%q in %q", q, corpus)
		}
	})
}
