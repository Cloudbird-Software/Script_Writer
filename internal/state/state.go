// Package state 实现 M1 状态内核：delta 协议 + apply + 三本台账
// （伏笔 open-loop、相遇、道具 instance）+ 时间轴 + 台词用量。
//
// 设计（issue #1 §B-1，本包是整个方案的支点）：
//
//	每集输入 = canon 快照 + 前情 + 未回收钩子表 + 本集申报 delta
//	每集输出 = 结构化 delta：新增实体/道具状态变化/开钩子/闭钩子(必须写 loop_id)/
//	           时间推进/卖点命中/情绪类型
//
// ApplyEp 只做台账层校验（守恒、状态机、单调性），不做正文文本检查（那是 gates 的职责）。
// 本包纯函数、无 IO（依赖方向见 docs/ARCHITECTURE.md：state ◀ gates ◀ passes）。
package state

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Cloudbird-Software/Script_Writer/internal/canon"
)

// Severity 取值。
const (
	SeverityError = "error" // 硬失败
	SeverityWarn  = "warn"  // 告警
)

// Gate 名（state 层自身产出的违规用 GateState；文本门禁由 gates 包使用其余常量）。
const (
	GateState          = "state"
	GateFormat         = "format"
	GateConsistency    = "consistency"
	GateRelationship   = "relationship"
	GateQuoteGrounding = "quote-grounding"
	GateHookPayoff     = "hook-payoff"
	GateLedgerClose    = "ledger-close"
)

// Violation 是全工具统一的违规报告单元（issue #1 §三 输出契约）。
type Violation struct {
	Gate     string `json:"gate"`
	Episode  int    `json:"episode"`
	Position string `json:"position"` // 句号/行号/字段路径等自由定位
	Expected string `json:"expected"`
	Actual   string `json:"actual"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

func (v Violation) String() string {
	return fmt.Sprintf("[E%d/%s/%s] %s（期望 %s，实际 %s）",
		v.Episode, v.Gate, v.Severity, v.Message, v.Expected, v.Actual)
}

// Delta 是一集的结构化申报（B-1 协议；强制模型申报"我关闭了哪个 loop_id"）。
type Delta struct {
	Meetings     []Meeting    `yaml:"meetings"`
	HooksOpened  []Hook       `yaml:"hooks_opened"`
	HooksClosed  []string     `yaml:"hooks_closed"`
	PropChanges  []PropChange `yaml:"prop_changes"`
	LineUses     []LineUse    `yaml:"line_uses"`
	SellingPoint string       `yaml:"selling_point"`
	Emotion      string       `yaml:"emotion"`
	Time         *TimeAdvance `yaml:"time"`
	NewFacts     []string     `yaml:"new_facts"`
	StateChanges []string     `yaml:"state_changes"`
}

// Meeting 一次首次相遇申报（entity id 对）。
type Meeting struct {
	A string `yaml:"a"`
	B string `yaml:"b"`
}

// Hook 是埋设的钩子/按钮。PickupKeywords 供钩子回收门做相邻集承接校验。
type Hook struct {
	LoopID         string   `yaml:"loop_id"`
	Priority       string   `yaml:"priority"` // P0 | P1 | P2
	Kind           string   `yaml:"kind"`     // hook | button
	Desc           string   `yaml:"desc"`
	PickupKeywords []string `yaml:"pickup_keywords"`
}

// PropChange 一次道具状态转移申报。
type PropChange struct {
	Instance string `yaml:"instance"`
	From     string `yaml:"from"`
	To       string `yaml:"to"`
}

// LineUse 一次台词资产使用申报。
type LineUse struct {
	Line  string `yaml:"line"`
	Count int    `yaml:"count"`
}

// TimeAdvance 一集的时间推进。
type TimeAdvance struct {
	Day    int    `yaml:"day"`
	Season string `yaml:"season"`
}

// Episode 是一集的完整输入：正文 + 元信息 + delta 申报。
type Episode struct {
	Ep          int    `yaml:"ep"`
	Text        string `yaml:"text"`
	TargetChars int    `yaml:"target_chars"`
	Finale      bool   `yaml:"finale"`
	Delta       Delta  `yaml:"delta"`
}

// Loop 是伏笔台账的一行。
type Loop struct {
	LoopID         string   `json:"loop_id"`
	Priority       string   `json:"priority"`
	Kind           string   `json:"kind"`
	Desc           string   `json:"desc"`
	PickupKeywords []string `json:"pickup_keywords"`
	OpenedEp       int      `json:"opened_ep"`
	ClosedEp       int      `json:"closed_ep"` // 0 = 未回收
}

// Open 报告该 loop 当前是否未回收。
func (l Loop) Open() bool { return l.ClosedEp == 0 }

// Ledger 是 apply 后的世界状态台账。
type Ledger struct {
	Loops      map[string]*Loop // loop_id → loop（含已回收，ClosedEp>0）
	MetPairs   map[string]int   // "A|B"（有序）→ 首次相遇集
	PropStates map[string]string
	LineTotals map[string]int
	LinePerEp  map[string]map[int]int // line id → ep → count
	LastDay    int
	AppliedEps []int
}

// NewLedger 基于 canon 初始化（道具 instance 从 initial_state 起步）。
func NewLedger(c *canon.Canon) *Ledger {
	l := &Ledger{
		Loops:      map[string]*Loop{},
		MetPairs:   map[string]int{},
		PropStates: map[string]string{},
		LineTotals: map[string]int{},
		LinePerEp:  map[string]map[int]int{},
		LastDay:    0,
	}
	for _, p := range c.Props {
		for _, inst := range p.Instances {
			l.PropStates[inst.ID] = inst.InitialState
		}
	}
	return l
}

// pairKey 返回有序对键，保证 A-B 与 B-A 同键。
func pairKey(a, b string) string {
	if a > b {
		a, b = b, a
	}
	return a + "|" + b
}

// Engine 顺序执行 apply 并保留每集后快照，供门禁/pass 消费。
type Engine struct {
	Canon   *canon.Canon
	Ledger  *Ledger
	history []Snapshot
}

// Snapshot 是某集 apply 之后的只读台账视图。
type Snapshot struct {
	Ep         int
	Loops      []Loop            // 按 OpenedEp 排序
	MetPairs   map[string]int    // 截至本集的首次相遇表
	PropStates map[string]string // 截至本集的道具状态
	LineTotals map[string]int
	LastDay    int
}

// NewEngine 创建状态机引擎。
func NewEngine(c *canon.Canon) *Engine {
	return &Engine{Canon: c, Ledger: NewLedger(c)}
}

// Snapshots 返回截至各集的快照序列（index i 对应第 i 个 apply 的集）。
func (e *Engine) Snapshots() []Snapshot { return e.history }

// ApplyEp 施加一集 delta，返回本集台账层违规；快照无论有无违规都会记录。
func (e *Engine) ApplyEp(ep Episode) []Violation {
	var vs []Violation
	l := e.Ledger
	c := e.Canon

	// 1. 伏笔：开钩子（id 唯一性）
	for _, h := range ep.Delta.HooksOpened {
		if h.LoopID == "" {
			vs = append(vs, Violation{Gate: GateState, Episode: ep.Ep, Position: "delta.hooks_opened",
				Severity: SeverityError, Message: "钩子 loop_id 为空"})
			continue
		}
		if prev, ok := l.Loops[h.LoopID]; ok {
			vs = append(vs, Violation{Gate: GateState, Episode: ep.Ep, Position: "delta.hooks_opened." + h.LoopID,
				Expected: "新 loop_id", Actual: h.LoopID, Severity: SeverityError,
				Message: fmt.Sprintf("loop_id 已存在（E%d 埋设，状态 %s）", prev.OpenedEp, loopState(prev))})
			continue
		}
		if h.Priority == "" {
			h.Priority = "P1"
		}
		if h.Kind == "" {
			h.Kind = "hook"
		}
		l.Loops[h.LoopID] = &Loop{
			LoopID:         h.LoopID,
			Priority:       h.Priority,
			Kind:           h.Kind,
			Desc:           h.Desc,
			PickupKeywords: h.PickupKeywords,
			OpenedEp:       ep.Ep,
		}
	}

	// 2. 伏笔：闭钩子（台账守恒：closed ⊆ opened，且不得重复回收）
	for _, id := range ep.Delta.HooksClosed {
		lp, ok := l.Loops[id]
		if !ok {
			vs = append(vs, Violation{Gate: GateState, Episode: ep.Ep, Position: "delta.hooks_closed." + id,
				Expected: "回收已埋设的 loop_id", Actual: id, Severity: SeverityError,
				Message: "回收了不存在的钩子（违反台账守恒 closed ⊆ opened）"})
			continue
		}
		if !lp.Open() {
			vs = append(vs, Violation{Gate: GateState, Episode: ep.Ep, Position: "delta.hooks_closed." + id,
				Expected: "未回收的 loop_id", Actual: id, Severity: SeverityError,
				Message: fmt.Sprintf("重复回收（E%d 已回收）", lp.ClosedEp)})
			continue
		}
		lp.ClosedEp = ep.Ep
	}

	// 3. 相遇：重复申报首次相遇 = 违规
	for _, m := range ep.Delta.Meetings {
		key := pairKey(m.A, m.B)
		if first, ok := l.MetPairs[key]; ok {
			vs = append(vs, Violation{Gate: GateState, Episode: ep.Ep, Position: "delta.meetings." + key,
				Expected: "新相遇对", Actual: key, Severity: SeverityError,
				Message: fmt.Sprintf("%s 与 %s 已在 E%d 相遇，不得重复申报首次相遇", m.A, m.B, first)})
			continue
		}
		l.MetPairs[key] = ep.Ep
	}

	// 4. 道具状态机
	for _, pc := range ep.Delta.PropChanges {
		var prop *canon.Prop
		var inst *canon.PropInstance
		for i := range c.Props {
			for j := range c.Props[i].Instances {
				if c.Props[i].Instances[j].ID == pc.Instance {
					prop, inst = &c.Props[i], &c.Props[i].Instances[j]
				}
			}
		}
		if prop == nil || inst == nil {
			vs = append(vs, Violation{Gate: GateState, Episode: ep.Ep, Position: "delta.prop_changes." + pc.Instance,
				Expected: "canon 中登记的 instance", Actual: pc.Instance, Severity: SeverityError,
				Message: "道具 instance 不存在"})
			continue
		}
		cur := l.PropStates[pc.Instance]
		if pc.From != cur {
			vs = append(vs, Violation{Gate: GateState, Episode: ep.Ep, Position: "delta.prop_changes." + pc.Instance,
				Expected: cur, Actual: pc.From, Severity: SeverityError,
				Message: "申报的起始状态与台账当前状态不符"})
			continue
		}
		legal := false
		for _, to := range prop.Transitions[pc.From] {
			if to == pc.To {
				legal = true
			}
		}
		if !legal {
			vs = append(vs, Violation{Gate: GateState, Episode: ep.Ep, Position: "delta.prop_changes." + pc.Instance,
				Expected: fmt.Sprintf("%v", prop.Transitions[pc.From]), Actual: pc.From + "→" + pc.To,
				Severity: SeverityError, Message: "道具状态转移不合法"})
			continue
		}
		l.PropStates[pc.Instance] = pc.To
	}

	// 5. 台词用量
	for _, lu := range ep.Delta.LineUses {
		var line *canon.Line
		for i := range c.Lines {
			if c.Lines[i].ID == lu.Line {
				line = &c.Lines[i]
			}
		}
		if line == nil {
			vs = append(vs, Violation{Gate: GateState, Episode: ep.Ep, Position: "delta.line_uses." + lu.Line,
				Expected: "canon 中登记的 line", Actual: lu.Line, Severity: SeverityError, Message: "台词资产不存在"})
			continue
		}
		if lu.Count < 1 {
			vs = append(vs, Violation{Gate: GateState, Episode: ep.Ep, Position: "delta.line_uses." + lu.Line,
				Expected: "≥1", Actual: fmt.Sprint(lu.Count), Severity: SeverityError, Message: "使用次数必须 ≥1"})
			continue
		}
		if l.LineTotals[lu.Line]+lu.Count > line.MaxUsesTotal {
			vs = append(vs, Violation{Gate: GateState, Episode: ep.Ep, Position: "delta.line_uses." + lu.Line,
				Expected: fmt.Sprintf("全剧 ≤%d 次", line.MaxUsesTotal),
				Actual:   fmt.Sprintf("累计将达 %d 次", l.LineTotals[lu.Line]+lu.Count),
				Severity: SeverityError, Message: "台词资产全剧用量超限"})
		}
		if lu.Count > line.MaxUsesPerEp {
			vs = append(vs, Violation{Gate: GateState, Episode: ep.Ep, Position: "delta.line_uses." + lu.Line,
				Expected: fmt.Sprintf("单集 ≤%d 次", line.MaxUsesPerEp), Actual: fmt.Sprint(lu.Count),
				Severity: SeverityError, Message: "台词资产单集用量超限"})
		}
		l.LineTotals[lu.Line] += lu.Count
		if l.LinePerEp[lu.Line] == nil {
			l.LinePerEp[lu.Line] = map[int]int{}
		}
		l.LinePerEp[lu.Line][ep.Ep] += lu.Count
	}

	// 6. 卖点命中：申报必须与排期一致
	if ep.Delta.SellingPoint != "" {
		expected := ""
		for _, s := range c.SellingPoints.Schedule {
			if s.Ep == ep.Ep {
				expected = s.Point
			}
		}
		if expected == "" {
			vs = append(vs, Violation{Gate: GateState, Episode: ep.Ep, Position: "delta.selling_point",
				Expected: "排期表中存在", Actual: ep.Delta.SellingPoint, Severity: SeverityError,
				Message: "本集不在卖点排期表中"})
		} else if expected != ep.Delta.SellingPoint {
			vs = append(vs, Violation{Gate: GateState, Episode: ep.Ep, Position: "delta.selling_point",
				Expected: expected, Actual: ep.Delta.SellingPoint, Severity: SeverityError,
				Message: "申报卖点与排期不一致"})
		}
	}

	// 7. 时间轴单调
	if t := ep.Delta.Time; t != nil && t.Day > 0 {
		if l.LastDay > 0 && t.Day < l.LastDay {
			vs = append(vs, Violation{Gate: GateState, Episode: ep.Ep, Position: "delta.time.day",
				Expected: fmt.Sprintf("≥%d", l.LastDay), Actual: fmt.Sprint(t.Day),
				Severity: SeverityError, Message: "序日倒退（时间轴必须单调不减）"})
		}
		l.LastDay = t.Day
	}

	l.AppliedEps = append(l.AppliedEps, ep.Ep)
	e.history = append(e.history, l.snapshot(ep.Ep))
	return vs
}

func loopState(l *Loop) string {
	if l.Open() {
		return "未回收"
	}
	return fmt.Sprintf("已回收于 E%d", l.ClosedEp)
}

func (l *Ledger) snapshot(ep int) Snapshot {
	s := Snapshot{
		Ep:         ep,
		Loops:      make([]Loop, 0, len(l.Loops)),
		MetPairs:   map[string]int{},
		PropStates: map[string]string{},
		LineTotals: map[string]int{},
		LastDay:    l.LastDay,
	}
	for _, lp := range l.Loops {
		s.Loops = append(s.Loops, *lp)
	}
	sort.Slice(s.Loops, func(i, j int) bool {
		if s.Loops[i].OpenedEp != s.Loops[j].OpenedEp {
			return s.Loops[i].OpenedEp < s.Loops[j].OpenedEp
		}
		return s.Loops[i].LoopID < s.Loops[j].LoopID
	})
	for k, v := range l.MetPairs {
		s.MetPairs[k] = v
	}
	for k, v := range l.PropStates {
		s.PropStates[k] = v
	}
	for k, v := range l.LineTotals {
		s.LineTotals[k] = v
	}
	return s
}

// OpenLoops 返回快照中仍未回收的钩子。
func (s Snapshot) OpenLoops() []Loop {
	var out []Loop
	for _, lp := range s.Loops {
		if lp.Open() {
			out = append(out, lp)
		}
	}
	return out
}

// MetBefore 报告 a、b 在快照时点是否已相遇，返回首次相遇集。
func (s Snapshot) MetBefore(a, b string) (int, bool) {
	ep, ok := s.MetPairs[pairKey(a, b)]
	return ep, ok
}

// FormatViolations 汇总打印（供 CLI/测试）。
func FormatViolations(vs []Violation) string {
	var b strings.Builder
	for _, v := range vs {
		b.WriteString(v.String())
		b.WriteString("\n")
	}
	return b.String()
}
