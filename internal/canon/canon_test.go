package canon

import (
	"path/filepath"
	"strings"
	"testing"

	rapid "pgregory.net/rapid"
)

// loadDemo 读取取材 issue #1 实际缺陷的演示 canon（宋驿）。
func loadDemo(t *testing.T) *Canon {
	t.Helper()
	c, err := Load(filepath.Join("testdata", "demo"))
	if err != nil {
		t.Fatalf("Load demo: %v", err)
	}
	return c
}

func TestLoadDemoAllTables(t *testing.T) {
	c := loadDemo(t)
	if len(c.Entities) != 5 || len(c.Props) != 1 || len(c.World.Rules) != 2 ||
		len(c.Lines) != 1 || len(c.SellingPoints.Schedule) != 5 || len(c.Timeline) != 5 {
		t.Fatalf("demo canon 数量不符: %+v", c)
	}
	if len(c.Meta.NameTitles) == 0 || len(c.Meta.PlaceSuffixes) == 0 {
		t.Fatal("meta.yaml 未加载")
	}
}

func TestLoadMissingFile(t *testing.T) {
	dir := t.TempDir()
	_, err := Load(dir)
	if err == nil || !strings.Contains(err.Error(), "缺少") {
		t.Fatalf("缺文件应报错，得到 %v", err)
	}
}

func TestValidateDemoClean(t *testing.T) {
	c := loadDemo(t)
	if ps := c.Validate(); len(ps) != 0 {
		t.Fatalf("演示 canon 应零问题，得到：%s", problems(ps))
	}
}

// 复现 issue #1 §A-3：没有代价条款的世界观不允许进入生成阶段。
func TestValidateWorldCostClauseRequired(t *testing.T) {
	c := loadDemo(t)
	c.World.Rules[1].CostClause = ""
	ps := c.Validate()
	if !containsField(ps, TableWorld, "cost_clause") {
		t.Fatalf("缺代价条款必须报问题，得到：%s", problems(ps))
	}
}

// 复现 P0-#1 前置：黑名单/白名单机制必须可被后续门禁消费。
func TestValidateEntityNameCollisions(t *testing.T) {
	c := loadDemo(t)
	c.Entities = append(c.Entities, Entity{
		ID: "FAKE", Type: "character", CanonicalName: "宁捕快", // 与既有实体重名
	})
	ps := c.Validate()
	if !containsTable(ps, TableEntities) {
		t.Fatalf("重名必须报问题，得到：%s", problems(ps))
	}
}

func TestValidatePropStateMachine(t *testing.T) {
	c := loadDemo(t)
	// 目标状态不在 states 中。
	c.Props[0].Transitions["持有"] = append(c.Props[0].Transitions["持有"], "飞升")
	ps := c.Validate()
	if !containsField(ps, TableProps, "transitions") {
		t.Fatalf("非法转移必须报问题，得到：%s", problems(ps))
	}
}

func TestValidateSellingPointConstraints(t *testing.T) {
	c := loadDemo(t)
	// 同卖点第三次出现 → 超 max_eps_per_point=2。
	c.SellingPoints.Schedule = append(c.SellingPoints.Schedule,
		ScheduledPoint{Ep: 6, Point: "热水淋浴", Category: "facility", Dimension: "胜负手"})
	// 且第三次维度与第二次（E5=胜负手）相同 → 违反换维度约束。
	ps := c.Validate()
	msgs := problems(ps)
	if !strings.Contains(msgs, "超过上限") || !strings.Contains(msgs, "换维度") {
		t.Fatalf("卖点约束必须报问题，得到：%s", msgs)
	}
}

func TestValidateServiceRatio(t *testing.T) {
	c := loadDemo(t)
	for i := range c.SellingPoints.Schedule {
		c.SellingPoints.Schedule[i].Category = "facility" // 服务占比 0%
	}
	ps := c.Validate()
	if !strings.Contains(problems(ps), "服务类卖点占比") {
		t.Fatalf("服务占比不足必须报问题，得到：%s", problems(ps))
	}
}

func TestValidateTimelineDayMonotonic(t *testing.T) {
	c := loadDemo(t)
	c.Timeline[4].Day = 3 // E5 day 倒退
	ps := c.Validate()
	if !containsField(ps, TableTimeline, "day") {
		t.Fatalf("序日倒退必须报问题，得到：%s", problems(ps))
	}
}

// PBT：任意实体列表中，只要存在两个不同 id 共享任一名称，Validate 必检出。
func TestPropertyDuplicateNamesAlwaysCaught(t *testing.T) {
	genName := rapid.Map(rapid.StringMatching(`[宁崔柳阿]{1}[白眉福快]{1}`), func(s string) string { return s })
	rapid.Check(t, func(t *rapid.T) {
		names := rapid.SliceOf(genName).Draw(t, "names")
		c := &Canon{World: World{Rules: []WorldRule{{ID: "W", Rule: "r", CostClause: "c"}}}}
		for i, n := range names {
			c.Entities = append(c.Entities, Entity{
				ID: string(rune('A'+i%26)) + string(rune('a'+i/26)), Type: "character", CanonicalName: n,
			})
		}
		dup := hasDupName(names)
		ps := c.Validate()
		if dup && !containsTable(ps, TableEntities) {
			t.Fatalf("存在重名 %v 但未检出：%s", names, problems(ps))
		}
	})
}

func hasDupName(names []string) bool {
	seen := map[string]bool{}
	for _, n := range names {
		if n == "" {
			continue
		}
		if seen[n] {
			return true
		}
		seen[n] = true
	}
	return false
}

func containsTable(ps []Problem, table string) bool {
	for _, p := range ps {
		if p.Table == table {
			return true
		}
	}
	return false
}

func containsField(ps []Problem, table, field string) bool {
	for _, p := range ps {
		if p.Table == table && strings.HasPrefix(p.Field, field) {
			return true
		}
	}
	return false
}

func problems(ps []Problem) string {
	var b strings.Builder
	for _, p := range ps {
		b.WriteString(p.String())
		b.WriteString("\n")
	}
	return b.String()
}

// 快照 demo canon 可完整加载且各表字段无静默丢失（yaml tag 拼写守卫）。
func TestYAMLTagsRoundTrip(t *testing.T) {
	c := loadDemo(t)
	if c.Entities[2].ForbiddenNames[0] != "渔捕快" {
		t.Fatal("forbidden_names 未解码（yaml tag 错误？）")
	}
	if c.Props[0].Instances[0].InitialState != "持有" {
		t.Fatal("initial_state 未解码")
	}
	if c.SellingPoints.Constraints.MinServiceRatio != 0.4 {
		t.Fatal("min_service_ratio 未解码")
	}
	if c.Timeline[0].TimeOfDay != "晚" {
		t.Fatal("time_of_day 未解码")
	}
}
