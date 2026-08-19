// Package canon 定义六张 canon 表（唯一事实来源）的 Go 类型、YAML 加载与结构校验。
//
// 六张表（issue #1 §A，散文 bible 升级为可机器校验的结构化 canon）：
//
//	entities.yaml        实体注册表：canonical_name + aliases 白名单 + forbidden_names 黑名单
//	props.yaml           道具本体 + 状态机（令牌 tiers/instances/transitions）
//	world.yaml           世界规则表：每条必须带非空 cost_clause（代价条款）
//	lines.yaml           台词资产表：slogan 归属/次数/语境
//	selling_points.yaml  卖点排期表：每集一对一排期 + 重复/服务占比约束
//	timeline.yaml        时间轴表：每集日期/季节/天气/时段
//
// 本包不依赖任何其它 internal 包（见 docs/ARCHITECTURE.md 依赖方向）。
package canon

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Table 名固定，Load 按此清单读取目录。
const (
	TableEntities      = "entities"
	TableProps         = "props"
	TableWorld         = "world"
	TableLines         = "lines"
	TableSellingPoints = "selling_points"
	TableTimeline      = "timeline"
	TableConfig        = "config" // 第七张表（M4）：门禁配置，可选
)

// Canon 是六张表 + 可选 config 的聚合根；对 YAML 目录 Load 后经 Validate 才可交给 state/gates 使用。
type Canon struct {
	Meta          Meta                `yaml:"meta"`
	Entities      []Entity            `yaml:"entities"`
	Props         []Prop              `yaml:"props"`
	World         World               `yaml:"world"`
	Lines         []Line              `yaml:"lines"`
	SellingPoints SellingPointsConfig `yaml:"selling_points"`
	Timeline      []TimelineEntry     `yaml:"timeline"`
	Config        Config              `yaml:"config"`
}

// Meta 承载漂移检测所需的全局词表（人称谓后缀、地名后缀）。
type Meta struct {
	NameTitles    []string `yaml:"name_titles"`    // 如 捕快/掌柜/娘子/爷 —— 一致性门用于拼出候选名
	PlaceSuffixes []string `yaml:"place_suffixes"` // 如 驿站/客栈/铺 —— 地名漂移候选后缀
}

// Entity 是实体注册表的一行：人物 / 地名 / 品牌 / 组织。
type Entity struct {
	ID               string     `yaml:"id"`
	Type             string     `yaml:"type"` // character | place | brand | org
	CanonicalName    string     `yaml:"canonical_name"`
	Aliases          []string   `yaml:"aliases"`
	ForbiddenNames   []string   `yaml:"forbidden_names"`
	RoleTimeline     []RoleSpan `yaml:"role_timeline"`
	AppearancePrompt string     `yaml:"appearance_prompt"`
	VoiceProfile     string     `yaml:"voice_profile"`
	KnowsSecret      bool       `yaml:"knows_secret"`
}

// RoleSpan 声明实体在某集数区间的角色（可多段）。
type RoleSpan struct {
	From int    `yaml:"from"`
	To   int    `yaml:"to"`
	Role string `yaml:"role"`
}

// Prop 是道具本体 + 状态机；instance 级台账由 state 包维护。
type Prop struct {
	ID          string              `yaml:"id"`
	Name        string              `yaml:"name"`
	Tiers       []string            `yaml:"tiers"`
	VisualRule  string              `yaml:"visual_rule"`
	IssueRule   string              `yaml:"issue_rule"`
	ReturnRule  string              `yaml:"return_rule"`
	States      []string            `yaml:"states"`
	Transitions map[string][]string `yaml:"transitions"`
	Instances   []PropInstance      `yaml:"instances"`
}

// PropInstance 是道具的具名实例（如 TOKEN_CUIBAI）。
type PropInstance struct {
	ID           string `yaml:"id"`
	Tier         string `yaml:"tier"`
	Holder       string `yaml:"holder"` // entity id
	InitialState string `yaml:"initial_state"`
}

// World 是世界规则表；issue #1：没有代价条款的世界观不允许进入生成阶段。
type World struct {
	Rules []WorldRule `yaml:"rules"`
}

// WorldRule 一条世界规则，CostClause 必须非空。
type WorldRule struct {
	ID         string   `yaml:"id"`
	Rule       string   `yaml:"rule"`
	CostClause string   `yaml:"cost_clause"`
	Knows      []string `yaml:"knows"` // 知情人 entity id 列表
}

// Line 是台词资产（slogan/口头禅）：归属、全剧与单集次数上限、语境约束。
type Line struct {
	ID              string   `yaml:"id"`
	Text            string   `yaml:"text"`
	Variants        []string `yaml:"variants"`
	Owner           string   `yaml:"owner"`
	MaxUsesTotal    int      `yaml:"max_uses_total"`
	MaxUsesPerEp    int      `yaml:"max_uses_per_ep"`
	RequiredContext string   `yaml:"required_context"`
}

// SellingPointsConfig 是卖点排期表 + 约束。
type SellingPointsConfig struct {
	Categories  []string           `yaml:"categories"` // 如 facility | service
	Constraints SellingConstraints `yaml:"constraints"`
	Schedule    []ScheduledPoint   `yaml:"schedule"`
}

// SellingConstraints 为卖点排期的硬约束（issue #1 §A-5）。
type SellingConstraints struct {
	MaxEpsPerPoint               int     `yaml:"max_eps_per_point"`
	SecondOccurrenceNewDimension bool    `yaml:"second_occurrence_new_dimension"`
	MinServiceRatio              float64 `yaml:"min_service_ratio"`
}

// ScheduledPoint 一集的主卖点排期。
type ScheduledPoint struct {
	Ep        int    `yaml:"ep"`
	Point     string `yaml:"point"`
	Category  string `yaml:"category"`
	Dimension string `yaml:"dimension"`
}

// TimelineEntry 一集的时间轴（Day 为序日，用于单调性校验）。
type TimelineEntry struct {
	Ep        int    `yaml:"ep"`
	Date      string `yaml:"date"`
	Day       int    `yaml:"day"`
	Season    string `yaml:"season"`
	Weather   string `yaml:"weather"`
	TimeOfDay string `yaml:"time_of_day"`
}

// Load 从目录读取六张 YAML 表；缺任一文件即报错（canon 必须六表齐备）。
// 每张文件的根键与 Canon 对应字段同名（如 entities.yaml 根键 entities:），
// 逐文件解码到同一聚合根即可聚合。
func Load(dir string) (*Canon, error) {
	c := &Canon{}
	for _, name := range []string{TableEntities, TableProps, TableWorld, TableLines, TableSellingPoints, TableTimeline} {
		raw, err := os.ReadFile(filepath.Join(dir, name+".yaml"))
		if err != nil {
			return nil, fmt.Errorf("canon: 缺少 %s.yaml: %w", name, err)
		}
		if err := yaml.Unmarshal(raw, c); err != nil {
			return nil, fmt.Errorf("canon: %s.yaml 解析失败: %w", name, err)
		}
	}
	// meta.yaml / config.yaml 可选（M4：config 缺省时各门禁用 WithDefaults 默认值）。
	for _, name := range []string{"meta", TableConfig} {
		if raw, err := os.ReadFile(filepath.Join(dir, name+".yaml")); err == nil {
			if err := yaml.Unmarshal(raw, c); err != nil {
				return nil, fmt.Errorf("canon: %s.yaml 解析失败: %w", name, err)
			}
		}
	}
	return c, nil
}

// EntityByID 返回 id → 实体的索引；重复 id 由 Validate 拦截，此处取首个。
func (c *Canon) EntityByID(id string) (Entity, bool) {
	for _, e := range c.Entities {
		if e.ID == id {
			return e, true
		}
	}
	return Entity{}, false
}

// KnownNames 返回所有合法名称（canonical + aliases），用于白名单比对。
func (c *Canon) KnownNames() map[string]string { // name → entity id
	out := make(map[string]string)
	for _, e := range c.Entities {
		out[e.CanonicalName] = e.ID
		for _, a := range e.Aliases {
			out[a] = e.ID
		}
	}
	return out
}
