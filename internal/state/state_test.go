package state

import (
	"fmt"
	"strings"
	"testing"

	rapid "pgregory.net/rapid"

	"github.com/Cloudbird-Software/Script_Writer/internal/canon"
)

func demoCanon(t *testing.T) *canon.Canon {
	t.Helper()
	c, err := canon.Load("../canon/testdata/demo")
	if err != nil {
		t.Fatalf("load demo canon: %v", err)
	}
	return c
}

func ep(n int, d Delta) Episode { return Episode{Ep: n, Delta: d} }

// 复现 P0-#4 类缺陷的机制：回收不存在的 loop_id（E30"过客有期"式凭空引文，
// 在台账层表现为关闭从未埋设的钩子）必须被守恒检查拦截。
func TestApplyUnknownHookClosed(t *testing.T) {
	e := NewEngine(demoCanon(t))
	vs := e.ApplyEp(ep(1, Delta{HooksClosed: []string{"L_GUEST_TOKEN"}}))
	if !hasMsg(vs, "台账守恒") {
		t.Fatalf("回收未知钩子必须违规，得到：%s", FormatViolations(vs))
	}
}

func TestApplyHookLifecycle(t *testing.T) {
	e := NewEngine(demoCanon(t))
	vs := e.ApplyEp(ep(1, Delta{HooksOpened: []Hook{{LoopID: "L1", Priority: "P0", PickupKeywords: []string{"木牌"}}}}))
	if len(vs) != 0 {
		t.Fatalf("E1 埋钩子不应违规：%s", FormatViolations(vs))
	}
	vs = append(vs, e.ApplyEp(ep(2, Delta{HooksClosed: []string{"L1"}}))...)
	vs = append(vs, e.ApplyEp(ep(3, Delta{HooksClosed: []string{"L1"}}))...) // 重复回收
	if !hasMsg(vs, "重复回收") {
		t.Fatalf("重复回收必须违规，得到：%s", FormatViolations(vs))
	}
	// 重复埋设同 id
	vs = e.ApplyEp(ep(4, Delta{HooksOpened: []Hook{{LoopID: "L1"}}}))
	if !hasMsg(vs, "loop_id 已存在") {
		t.Fatalf("重复埋设必须违规，得到：%s", FormatViolations(vs))
	}
}

// 复现 P0-#3 机制：E12 崔白"初次相识"类缺陷在台账层 = 重复申报首次相遇。
func TestApplyDuplicateMeeting(t *testing.T) {
	e := NewEngine(demoCanon(t))
	var vs []Violation
	vs = append(vs, e.ApplyEp(ep(1, Delta{Meetings: []Meeting{{A: "CUI_BAI", B: "LIU_QINGMEI"}}}))...)
	vs = append(vs, e.ApplyEp(ep(12, Delta{Meetings: []Meeting{{A: "LIU_QINGMEI", B: "CUI_BAI"}}}))...)
	if !hasMsg(vs, "重复申报首次相遇") {
		t.Fatalf("重复相遇必须违规，得到：%s", FormatViolations(vs))
	}
	if _, ok := e.Snapshots()[1].MetBefore("CUI_BAI", "LIU_QINGMEI"); !ok {
		t.Fatal("快照必须记录 E1 相遇（供关系门消费）")
	}
}

// 令牌状态机：E12 抵押 → E24 赎回（合法随机游走）。
func TestApplyPropStateMachine(t *testing.T) {
	e := NewEngine(demoCanon(t))
	var vs []Violation
	vs = append(vs, e.ApplyEp(ep(12, Delta{PropChanges: []PropChange{{Instance: "TOKEN_CUIBAI", From: "持有", To: "抵押"}}}))...)
	if len(vs) != 0 {
		t.Fatalf("合法转移不应违规：%s", FormatViolations(vs))
	}
	vs = e.ApplyEp(ep(13, Delta{PropChanges: []PropChange{{Instance: "TOKEN_CUIBAI", From: "库存", To: "持有"}}}))
	if !hasMsg(vs, "起始状态与台账") {
		t.Fatalf("错误起始状态必须违规，得到：%s", FormatViolations(vs))
	}
	vs = e.ApplyEp(ep(14, Delta{PropChanges: []PropChange{{Instance: "TOKEN_CUIBAI", From: "抵押", To: "库存"}}}))
	if !hasMsg(vs, "状态转移不合法") {
		t.Fatalf("非法转移必须违规，得到：%s", FormatViolations(vs))
	}
}

func TestApplyLineUseLimits(t *testing.T) {
	e := NewEngine(demoCanon(t))
	vs := e.ApplyEp(ep(1, Delta{LineUses: []LineUse{{Line: "SLOGAN_1", Count: 2}}})) // 单集上限 1
	if !hasMsg(vs, "单集用量超限") {
		t.Fatalf("单集超限必须违规，得到：%s", FormatViolations(vs))
	}
	for i := 2; i <= 8; i++ {
		e.ApplyEp(ep(i, Delta{LineUses: []LineUse{{Line: "SLOGAN_1", Count: 1}}}))
	}
	vs = e.ApplyEp(ep(9, Delta{LineUses: []LineUse{{Line: "SLOGAN_1", Count: 1}}})) // 全剧上限 8
	if !hasMsg(vs, "全剧用量超限") {
		t.Fatalf("全剧超限必须违规，得到：%s", FormatViolations(vs))
	}
}

func TestApplySellingPointSchedule(t *testing.T) {
	e := NewEngine(demoCanon(t))
	vs := e.ApplyEp(ep(1, Delta{SellingPoint: "隐私尊重"})) // E1 排期是 热水淋浴
	if !hasMsg(vs, "与排期不一致") {
		t.Fatalf("卖点错配必须违规，得到：%s", FormatViolations(vs))
	}
	vs = e.ApplyEp(ep(99, Delta{SellingPoint: "热水淋浴"}))
	if !hasMsg(vs, "不在卖点排期表") {
		t.Fatalf("无排期集申报卖点必须违规，得到：%s", FormatViolations(vs))
	}
}

func TestApplyDayMonotonic(t *testing.T) {
	e := NewEngine(demoCanon(t))
	var vs []Violation
	vs = append(vs, e.ApplyEp(ep(1, Delta{Time: &TimeAdvance{Day: 5}}))...)
	vs = append(vs, e.ApplyEp(ep(2, Delta{Time: &TimeAdvance{Day: 3}}))...)
	if !hasMsg(vs, "序日倒退") {
		t.Fatalf("序日倒退必须违规，得到：%s", FormatViolations(vs))
	}
}

// M4 弧线台账：单调不倒退 + 单步 ≤+1（复现"开局即内部人/23集老板"类缺陷的台账面）。
func TestApplyArcMonotonic(t *testing.T) {
	e := NewEngine(demoCanon(t))
	var vs []Violation
	vs = append(vs, e.ApplyEp(ep(1, Delta{Arc: &ArcAdvance{Level: 0, Cost: "初来乍到，只能旁观"}}))...)
	if len(vs) != 0 {
		t.Fatalf("首次申报 L0 不应违规：%s", FormatViolations(vs))
	}
	vs = append(vs, e.ApplyEp(ep(5, Delta{Arc: &ArcAdvance{Level: 1, Cost: "替店里挡了一场官差"}}))...)
	if len(vs) != 0 {
		t.Fatalf("L0→L1 不应违规：%s", FormatViolations(vs))
	}
	vs = append(vs, e.ApplyEp(ep(6, Delta{Arc: &ArcAdvance{Level: 0}}))...) // 倒退
	vs = append(vs, e.ApplyEp(ep(7, Delta{Arc: &ArcAdvance{Level: 4}}))...) // 跳级
	if !hasMsg(vs, "权限等级倒退") {
		t.Fatalf("等级倒退必须违规，得到：%s", FormatViolations(vs))
	}
	if !hasMsg(vs, "权限等级跳级") {
		t.Fatalf("等级跳级必须违规，得到：%s", FormatViolations(vs))
	}
	if got := e.Snapshots()[1].ArcLevel; got != 1 {
		t.Fatalf("快照应记录 ArcLevel=1，得到 %d", got)
	}
}

// M4 群演登记：Crowd 申报累计进台账，供可拍性门做全剧配额判定。
func TestApplyCrowdLedger(t *testing.T) {
	e := NewEngine(demoCanon(t))
	e.ApplyEp(ep(1, Delta{Crowd: true, Scenes: []string{"大堂", "后厨"}}))
	e.ApplyEp(ep(2, Delta{Crowd: true}))
	if got := len(e.Ledger.CrowdEps); got != 2 {
		t.Fatalf("群演场面应累计 2 集，得到 %d", got)
	}
	if s := e.Snapshots()[0]; len(s.Scenes) != 2 || !s.Crowd {
		t.Fatalf("快照应携带本集 scenes/crowd：%v/%v", s.Scenes, s.Crowd)
	}
}

// PBT-1 台账守恒：对任意随机开/闭序列，凡闭了未开（或重复闭）的 id，
// ApplyEp 必产出至少一条守恒类违规；只闭已开且未闭的 id 则零守恒违规。
func TestPropertyLedgerConservation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		ids := rapid.SliceOfN(rapid.StringMatching(`L[0-9]{1,2}`), 1, 6).Draw(t, "ids")
		opens := rapid.SliceOf(rapid.IntRange(0, len(ids)-1)).Draw(t, "openIdx")
		closes := rapid.SliceOf(rapid.IntRange(0, len(ids)-1)).Draw(t, "closeIdx")
		c := demoCanon(&testing.T{})
		e := NewEngine(c)
		opened := map[string]bool{}
		closedAgain := false
		var vs []Violation
		d1 := Delta{}
		for _, i := range opens {
			if !opened[ids[i]] {
				d1.HooksOpened = append(d1.HooksOpened, Hook{LoopID: ids[i]})
				opened[ids[i]] = true
			}
		}
		vs = append(vs, e.ApplyEp(ep(1, d1))...)
		d2 := Delta{}
		seenClose := map[string]bool{}
		for _, i := range closes {
			d2.HooksClosed = append(d2.HooksClosed, ids[i])
			if !opened[ids[i]] || seenClose[ids[i]] {
				closedAgain = true
			}
			seenClose[ids[i]] = true
		}
		vs = append(vs, e.ApplyEp(ep(2, d2))...)
		bad := false
		for _, v := range vs {
			if strings.Contains(v.Message, "台账守恒") || strings.Contains(v.Message, "重复回收") {
				bad = true
			}
		}
		if closedAgain && !bad {
			t.Fatalf("闭了非法 id 但未检出守恒违规：opens=%v closes=%v → %s", opens, closes, FormatViolations(vs))
		}
	})
}

// PBT-2 状态机随机游走：只走 canon 声明的合法转移 → 零道具违规。
func TestPropertyPropWalkValid(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		c := demoCanon(&testing.T{})
		e := NewEngine(c)
		p := c.Props[0]
		inst := p.Instances[0]
		cur := inst.InitialState
		for epN := 1; epN <= 6; epN++ {
			tos := p.Transitions[cur]
			if len(tos) == 0 {
				break
			}
			next := rapid.SampledFrom(tos).Draw(t, fmt.Sprintf("next%d", epN))
			vs := e.ApplyEp(ep(epN, Delta{PropChanges: []PropChange{{Instance: inst.ID, From: cur, To: next}}}))
			for _, v := range vs {
				if v.Position == "delta.prop_changes."+inst.ID {
					t.Fatalf("合法转移被误报：%s", v)
				}
			}
			cur = next
		}
	})
}

// PBT-3 守恒不变量（模型侧）：任意合法 apply 序列后，ClosedEp>0 的 loop 必有 OpenedEp>0
// 且 OpenedEp ≤ ClosedEp（时间不可倒流）。
func TestPropertyClosedSubsetOpened(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		c := demoCanon(&testing.T{})
		e := NewEngine(c)
		n := rapid.IntRange(1, 10).Draw(t, "eps")
		var openIDs []string
		for i := 1; i <= n; i++ {
			d := Delta{}
			if rapid.Bool().Draw(t, fmt.Sprintf("open%d", i)) {
				id := fmt.Sprintf("L%d", i)
				d.HooksOpened = []Hook{{LoopID: id}}
				openIDs = append(openIDs, id)
			}
			if len(openIDs) > 0 && rapid.Bool().Draw(t, fmt.Sprintf("close%d", i)) {
				d.HooksClosed = []string{openIDs[0]}
			}
			e.ApplyEp(ep(i, d))
		}
		snap := e.Snapshots()[len(e.Snapshots())-1]
		for _, lp := range snap.Loops {
			if !lp.Open() {
				if lp.OpenedEp == 0 || lp.OpenedEp > lp.ClosedEp {
					t.Fatalf("守恒被破坏：%+v", lp)
				}
			}
		}
	})
}

func hasMsg(vs []Violation, sub string) bool {
	for _, v := range vs {
		if strings.Contains(v.Message, sub) {
			return true
		}
	}
	return false
}
