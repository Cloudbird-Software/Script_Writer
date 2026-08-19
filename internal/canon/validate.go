package canon

import (
	"fmt"
	"sort"
)

// Problem 是一条 canon 结构问题；M2 之前的唯一"违规"形态（gate=canon）。
type Problem struct {
	Table   string // 六表之一
	ID      string // 相关条目 id（可为空）
	Field   string // 字段名
	Message string
}

func (p Problem) String() string {
	loc := p.Table
	if p.ID != "" {
		loc += "." + p.ID
	}
	if p.Field != "" {
		loc += "." + p.Field
	}
	return fmt.Sprintf("%s: %s", loc, p.Message)
}

var entityTypes = map[string]bool{"character": true, "place": true, "brand": true, "org": true}
var seasons = map[string]bool{"spring": true, "summer": true, "autumn": true, "winter": true}

// Validate 对六张表 + 可选 config 做全量结构校验，返回问题列表；空列表 = canon 可用。
func (c *Canon) Validate() []Problem {
	var ps []Problem
	ps = append(ps, c.validateEntities()...)
	ps = append(ps, c.validateProps()...)
	ps = append(ps, c.validateWorld()...)
	ps = append(ps, c.validateLines()...)
	ps = append(ps, c.validateSellingPoints()...)
	ps = append(ps, c.validateTimeline()...)
	ps = append(ps, c.validateConfig()...)
	return ps
}

func (c *Canon) validateConfig() []Problem {
	var ps []Problem
	cfg := c.Config
	// 只有显式配置了才校验；零值 config 合法（门禁走 WithDefaults）。
	if cfg.Arc.Character != "" {
		if ch, ok := c.EntityByID(cfg.Arc.Character); !ok {
			ps = append(ps, Problem{Table: TableConfig, Field: "arc.character",
				Message: fmt.Sprintf("%q 不在 entities 中", cfg.Arc.Character)})
		} else if ch.Type != "character" {
			ps = append(ps, Problem{Table: TableConfig, Field: "arc.character",
				Message: fmt.Sprintf("%q 不是 character", cfg.Arc.Character)})
		}
	}
	if cfg.Format.CharsTolerance < 0 || cfg.Format.CharsTolerance > 1 {
		ps = append(ps, Problem{Table: TableConfig, Field: "format.chars_tolerance",
			Message: "必须在 (0,1] 区间"})
	}
	if cfg.HookPayoff.WarnAfterEps > cfg.HookPayoff.FailAfterEps && cfg.HookPayoff.FailAfterEps > 0 {
		ps = append(ps, Problem{Table: TableConfig, Field: "hook_payoff",
			Message: fmt.Sprintf("warn_after_eps(%d) 不得大于 fail_after_eps(%d)",
				cfg.HookPayoff.WarnAfterEps, cfg.HookPayoff.FailAfterEps)})
	}
	seenCat := map[string]bool{}
	for _, cat := range cfg.Compliance.Categories {
		if cat.ID == "" {
			ps = append(ps, Problem{Table: TableConfig, Field: "compliance.categories",
				Message: "存在空 id"})
			continue
		}
		if seenCat[cat.ID] {
			ps = append(ps, Problem{Table: TableConfig, Field: "compliance.categories." + cat.ID,
				Message: "id 重复"})
		}
		seenCat[cat.ID] = true
		if cat.Level != ComplianceHard && cat.Level != ComplianceFlag {
			ps = append(ps, Problem{Table: TableConfig, Field: "compliance.categories." + cat.ID,
				Message: fmt.Sprintf("level %q 不在 {hard,flag}", cat.Level)})
		}
		if len(cat.Patterns) == 0 {
			ps = append(ps, Problem{Table: TableConfig, Field: "compliance.categories." + cat.ID,
				Message: "patterns 为空"})
		}
	}
	return ps
}

func (c *Canon) validateEntities() []Problem {
	var ps []Problem
	seenID := map[string]bool{}
	seenName := map[string]string{} // name → entity id
	for _, e := range c.Entities {
		if e.ID == "" {
			ps = append(ps, Problem{Table: TableEntities, Message: "存在空 id"})
			continue
		}
		if seenID[e.ID] {
			ps = append(ps, Problem{Table: TableEntities, ID: e.ID, Message: "id 重复"})
		}
		seenID[e.ID] = true
		if !entityTypes[e.Type] {
			ps = append(ps, Problem{Table: TableEntities, ID: e.ID, Field: "type",
				Message: fmt.Sprintf("type %q 不在 {character,place,brand,org}", e.Type)})
		}
		if e.CanonicalName == "" {
			ps = append(ps, Problem{Table: TableEntities, ID: e.ID, Field: "canonical_name", Message: "为空"})
		}
		forbidden := map[string]bool{}
		for _, f := range e.ForbiddenNames {
			forbidden[f] = true
		}
		for _, a := range e.Aliases {
			if a == e.CanonicalName {
				ps = append(ps, Problem{Table: TableEntities, ID: e.ID, Field: "aliases",
					Message: fmt.Sprintf("别名 %q 与 canonical_name 重复", a)})
			}
			if forbidden[a] {
				ps = append(ps, Problem{Table: TableEntities, ID: e.ID, Field: "aliases",
					Message: fmt.Sprintf("别名 %q 同时出现在 forbidden_names", a)})
			}
		}
		// 名称全局唯一：两个实体不得共享任何名字（含别名）。
		for _, n := range append([]string{e.CanonicalName}, e.Aliases...) {
			if n == "" {
				continue
			}
			if owner, ok := seenName[n]; ok && owner != e.ID {
				ps = append(ps, Problem{Table: TableEntities, ID: e.ID, Field: "names",
					Message: fmt.Sprintf("名称 %q 与实体 %s 冲突（名称必须全局唯一）", n, owner)})
			}
			seenName[n] = e.ID
		}
		for i, r := range e.RoleTimeline {
			if r.From > r.To {
				ps = append(ps, Problem{Table: TableEntities, ID: e.ID, Field: fmt.Sprintf("role_timeline[%d]", i),
					Message: fmt.Sprintf("from(%d) > to(%d)", r.From, r.To)})
			}
		}
	}
	return ps
}

func (c *Canon) validateProps() []Problem {
	var ps []Problem
	seenInst := map[string]bool{}
	for _, p := range c.Props {
		if p.ID == "" {
			ps = append(ps, Problem{Table: TableProps, Message: "存在空 id"})
			continue
		}
		if len(p.States) == 0 {
			ps = append(ps, Problem{Table: TableProps, ID: p.ID, Field: "states", Message: "为空（状态机必须显式声明状态集）"})
		}
		stateSet := map[string]bool{}
		for _, s := range p.States {
			stateSet[s] = true
		}
		for from, tos := range p.Transitions {
			if !stateSet[from] {
				ps = append(ps, Problem{Table: TableProps, ID: p.ID, Field: "transitions",
					Message: fmt.Sprintf("源状态 %q 不在 states 中", from)})
			}
			for _, to := range tos {
				if !stateSet[to] {
					ps = append(ps, Problem{Table: TableProps, ID: p.ID, Field: "transitions",
						Message: fmt.Sprintf("目标状态 %q 不在 states 中", to)})
				}
			}
		}
		tierSet := map[string]bool{}
		for _, t := range p.Tiers {
			tierSet[t] = true
		}
		for _, inst := range p.Instances {
			if inst.ID == "" {
				ps = append(ps, Problem{Table: TableProps, ID: p.ID, Field: "instances", Message: "存在空 instance id"})
				continue
			}
			if seenInst[inst.ID] {
				ps = append(ps, Problem{Table: TableProps, ID: p.ID, Field: "instances",
					Message: fmt.Sprintf("instance id %q 重复", inst.ID)})
			}
			seenInst[inst.ID] = true
			if len(p.Tiers) > 0 && !tierSet[inst.Tier] {
				ps = append(ps, Problem{Table: TableProps, ID: p.ID, Field: "instances." + inst.ID,
					Message: fmt.Sprintf("tier %q 不在 tiers 中", inst.Tier)})
			}
			if holder, ok := c.EntityByID(inst.Holder); !ok {
				ps = append(ps, Problem{Table: TableProps, ID: p.ID, Field: "instances." + inst.ID,
					Message: fmt.Sprintf("holder %q 不在 entities 中", inst.Holder)})
			} else if holder.Type != "character" {
				ps = append(ps, Problem{Table: TableProps, ID: p.ID, Field: "instances." + inst.ID,
					Message: fmt.Sprintf("holder %q 不是 character", inst.Holder)})
			}
			if !stateSet[inst.InitialState] {
				ps = append(ps, Problem{Table: TableProps, ID: p.ID, Field: "instances." + inst.ID,
					Message: fmt.Sprintf("initial_state %q 不在 states 中", inst.InitialState)})
			}
		}
	}
	return ps
}

func (c *Canon) validateWorld() []Problem {
	var ps []Problem
	if len(c.World.Rules) == 0 {
		ps = append(ps, Problem{Table: TableWorld, Message: "世界规则表为空（世界观未声明即不允许生成，issue #1 §A-3）"})
		return ps
	}
	for _, r := range c.World.Rules {
		if r.ID == "" {
			ps = append(ps, Problem{Table: TableWorld, Message: "存在空 id"})
		}
		if r.Rule == "" {
			ps = append(ps, Problem{Table: TableWorld, ID: r.ID, Field: "rule", Message: "为空"})
		}
		// issue #1：没有代价条款的世界观，一律不允许进入生成阶段。
		if r.CostClause == "" {
			ps = append(ps, Problem{Table: TableWorld, ID: r.ID, Field: "cost_clause",
				Message: "为空——每条世界规则必须声明代价条款"})
		}
		for _, k := range r.Knows {
			if _, ok := c.EntityByID(k); !ok {
				ps = append(ps, Problem{Table: TableWorld, ID: r.ID, Field: "knows",
					Message: fmt.Sprintf("知情人 %q 不在 entities 中", k)})
			}
		}
	}
	return ps
}

func (c *Canon) validateLines() []Problem {
	var ps []Problem
	for _, l := range c.Lines {
		if l.ID == "" {
			ps = append(ps, Problem{Table: TableLines, Message: "存在空 id"})
			continue
		}
		if l.Text == "" {
			ps = append(ps, Problem{Table: TableLines, ID: l.ID, Field: "text", Message: "为空"})
		}
		if owner, ok := c.EntityByID(l.Owner); !ok {
			ps = append(ps, Problem{Table: TableLines, ID: l.ID, Field: "owner",
				Message: fmt.Sprintf("owner %q 不在 entities 中", l.Owner)})
		} else if owner.Type != "character" {
			ps = append(ps, Problem{Table: TableLines, ID: l.ID, Field: "owner",
				Message: fmt.Sprintf("owner %q 不是 character", l.Owner)})
		}
		if l.MaxUsesTotal < 1 {
			ps = append(ps, Problem{Table: TableLines, ID: l.ID, Field: "max_uses_total", Message: "必须 ≥1"})
		}
		if l.MaxUsesPerEp < 1 {
			ps = append(ps, Problem{Table: TableLines, ID: l.ID, Field: "max_uses_per_ep", Message: "必须 ≥1"})
		}
	}
	return ps
}

func (c *Canon) validateSellingPoints() []Problem {
	var ps []Problem
	sp := c.SellingPoints
	catSet := map[string]bool{}
	for _, cat := range sp.Categories {
		catSet[cat] = true
	}
	if len(sp.Categories) > 0 && !catSet["service"] {
		ps = append(ps, Problem{Table: TableSellingPoints, Field: "categories",
			Message: "必须包含 service 类别（服务类卖点占比约束依赖它）"})
	}
	if sp.Constraints.MaxEpsPerPoint < 1 {
		ps = append(ps, Problem{Table: TableSellingPoints, Field: "constraints.max_eps_per_point", Message: "必须 ≥1"})
	}
	seenEp := map[int]bool{}
	byPoint := map[string][]ScheduledPoint{}
	for _, s := range sp.Schedule {
		if seenEp[s.Ep] {
			ps = append(ps, Problem{Table: TableSellingPoints, Field: "schedule",
				Message: fmt.Sprintf("第 %d 集重复排期（每集主卖点一对一）", s.Ep)})
		}
		seenEp[s.Ep] = true
		if len(sp.Categories) > 0 && !catSet[s.Category] {
			ps = append(ps, Problem{Table: TableSellingPoints, Field: "schedule",
				Message: fmt.Sprintf("E%d 类别 %q 不在 categories 中", s.Ep, s.Category)})
		}
		if s.Point == "" {
			ps = append(ps, Problem{Table: TableSellingPoints, Field: "schedule",
				Message: fmt.Sprintf("E%d point 为空", s.Ep)})
		}
		byPoint[s.Point] = append(byPoint[s.Point], s)
	}
	maxEps := sp.Constraints.MaxEpsPerPoint
	if maxEps < 1 {
		maxEps = 1 // 已另行报问题；防除零/误报
	}
	for point, eps := range byPoint {
		if len(eps) > maxEps {
			ps = append(ps, Problem{Table: TableSellingPoints, Field: "schedule",
				Message: fmt.Sprintf("卖点 %q 排了 %d 集，超过上限 %d", point, len(eps), maxEps)})
		}
		if c.SellingPoints.Constraints.SecondOccurrenceNewDimension && len(eps) >= 2 {
			sort.Slice(eps, func(i, j int) bool { return eps[i].Ep < eps[j].Ep })
			for i := 1; i < len(eps); i++ {
				if eps[i].Dimension == eps[i-1].Dimension && eps[i].Dimension != "" {
					ps = append(ps, Problem{Table: TableSellingPoints, Field: "schedule",
						Message: fmt.Sprintf("卖点 %q 第 %d 次出现（E%d）维度 %q 与上次相同——必须换维度",
							point, i+1, eps[i].Ep, eps[i].Dimension)})
				}
			}
		}
	}
	if n := len(sp.Schedule); n > 0 && sp.Constraints.MinServiceRatio > 0 {
		service := 0
		for _, s := range sp.Schedule {
			if s.Category == "service" {
				service++
			}
		}
		if ratio := float64(service) / float64(n); ratio < sp.Constraints.MinServiceRatio {
			ps = append(ps, Problem{Table: TableSellingPoints, Field: "schedule",
				Message: fmt.Sprintf("服务类卖点占比 %.0f%% 低于下限 %.0f%%", ratio*100, sp.Constraints.MinServiceRatio*100)})
		}
	}
	return ps
}

func (c *Canon) validateTimeline() []Problem {
	var ps []Problem
	seenEp := map[int]bool{}
	entries := append([]TimelineEntry(nil), c.Timeline...)
	sort.Slice(entries, func(i, j int) bool { return entries[i].Ep < entries[j].Ep })
	for _, t := range entries {
		if seenEp[t.Ep] {
			ps = append(ps, Problem{Table: TableTimeline, Field: "ep",
				Message: fmt.Sprintf("第 %d 集时间轴重复", t.Ep)})
		}
		seenEp[t.Ep] = true
		if t.Season != "" && !seasons[t.Season] {
			ps = append(ps, Problem{Table: TableTimeline, Field: "season",
				Message: fmt.Sprintf("E%d season %q 不在 {spring,summer,autumn,winter}", t.Ep, t.Season)})
		}
	}
	prevDay := -1
	for _, t := range entries {
		if t.Day > 0 {
			if prevDay > 0 && t.Day < prevDay {
				ps = append(ps, Problem{Table: TableTimeline, Field: "day",
					Message: fmt.Sprintf("E%d day=%d 小于前条 %d（序日必须单调不减）", t.Ep, t.Day, prevDay)})
			}
			prevDay = t.Day
		}
	}
	return ps
}
