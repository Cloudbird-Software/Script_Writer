package canon

import (
	"os"
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

// M4：config.yaml 可选——六表 canon（无 config）合法，门禁走 WithDefaults。
func TestConfigOptionalAndDefaults(t *testing.T) {
	c := loadDemo(t)
	if ps := c.Validate(); len(ps) != 0 {
		t.Fatalf("无 config 的 canon 应合法，得到：%s", problems(ps))
	}
	d := c.Config.WithDefaults()
	if d.Format.CharsTolerance != 0.10 || d.HookPayoff.WarnAfterEps != 6 ||
		d.HookPayoff.FailAfterEps != 10 || d.Emotion.MaxStreak != 3 ||
		d.Producibility.MaxNamedCharsPerEp != 5 || d.Producibility.MaxScenesPerEp != 3 {
		t.Fatalf("默认值不符 issue #1 §B-2 推荐值: %+v", d)
	}
	// 默认词表必须直接覆盖 issue #1 缺陷清单里已发生的项。
	if _, ok := d.Hygiene.Typos["愣"]; !ok {
		t.Fatal("默认错别字表应含 issue #1 实际缺陷项（愣→怔）")
	}
	if !containsStr(d.Producibility.CameraTerms, "镜头一抬") {
		t.Fatal("默认镜头语言表应含 issue #1 实际泄漏项（镜头一抬）")
	}
	found := false
	for _, cat := range d.Compliance.Categories {
		if cat.ID == "official-favor" && containsStr(cat.Patterns, "铺条官路") {
			found = true
		}
	}
	if !found {
		t.Fatal("默认合规词表应含 issue #1 §3-5 官商往来项（铺条官路）")
	}
}

// M4：显式配置的 config 会被结构校验（arc 指向不存在实体 / level 非法 / 阈值反置）。
func TestValidateConfigProblems(t *testing.T) {
	c := loadDemo(t)
	c.Config = Config{
		Arc:        ArcConfig{Character: "NOBODY"},
		Format:     FormatConfig{CharsTolerance: 1.5},
		HookPayoff: HookPayoffConfig{WarnAfterEps: 20, FailAfterEps: 10},
		Compliance: ComplianceConfig{Categories: []ComplianceCategory{
			{ID: "x", Level: "soft", Patterns: []string{"foo"}},
		}},
	}
	ps := c.Validate()
	for _, want := range []string{"arc.character", "format.chars_tolerance", "hook_payoff", "compliance"} {
		if !containsField(ps, TableConfig, want) {
			t.Fatalf("config 问题 %q 未检出，得到：%s", want, problems(ps))
		}
	}
}

// M4：config.yaml 从目录加载的 yaml tag round-trip（复制 demo 目录 + 注入 config）。
func TestLoadConfigYAML(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{TableEntities, TableProps, TableWorld, TableLines, TableSellingPoints, TableTimeline, "meta"} {
		raw, err := os.ReadFile(filepath.Join("testdata", "demo", name+".yaml"))
		if err != nil {
			t.Fatalf("read demo %s.yaml: %v", name, err)
		}
		if err := os.WriteFile(filepath.Join(dir, name+".yaml"), raw, 0o644); err != nil {
			t.Fatalf("copy %s.yaml: %v", name, err)
		}
	}
	cfg := `config:
  arc:
    character: LIU_QINGMEI
    max_level: 5
    min_eps_per_step: 5
  emotion:
    types: [惊叹, 危机]
    max_streak: 3
  hygiene:
    typos: {愣: 怔}
    garbled_patterns: [暖幢栋]
  compliance:
    categories:
      - id: official-favor
        level: hard
        patterns: [铺条官路]
`
	if err := os.WriteFile(filepath.Join(dir, TableConfig+".yaml"), []byte(cfg), 0o644); err != nil {
		t.Fatalf("write config.yaml: %v", err)
	}
	c, err := Load(dir)
	if err != nil {
		t.Fatalf("Load with config: %v", err)
	}
	if c.Config.Arc.Character != "LIU_QINGMEI" || c.Config.Arc.MinEpsPerStep != 5 {
		t.Fatalf("arc 段未解码: %+v", c.Config.Arc)
	}
	if c.Config.Emotion.MaxStreak != 3 || len(c.Config.Emotion.Types) != 2 {
		t.Fatalf("emotion 段未解码: %+v", c.Config.Emotion)
	}
	if c.Config.Hygiene.Typos["愣"] != "怔" || len(c.Config.Hygiene.GarbledPatterns) != 1 {
		t.Fatalf("hygiene 段未解码: %+v", c.Config.Hygiene)
	}
	if got := c.Config.Compliance.Categories[0]; got.ID != "official-favor" || got.Level != ComplianceHard {
		t.Fatalf("compliance 段未解码: %+v", got)
	}
	if ps := c.Validate(); len(ps) != 0 {
		t.Fatalf("合法 config 应零问题，得到：%s", problems(ps))
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

func containsStr(xs []string, want string) bool {
	for _, x := range xs {
		if x == want {
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
