package canon

// M4 门禁配置（第七张表 config.yaml，issue #1 §B-2 十二道门 + 两道软门的可调参数）。
//
// 设计原则（ADR-0002）：
//   - 阈值与词表是资产（随剧目变化），不是代码常量——进 canon、随仓库版本化；
//   - config.yaml 可选：缺省时各段零值，经 WithDefaults() 填充内置默认，
//     保证旧 canon 目录（六张表）无需改动即可继续使用；
//   - 词表刻意从 issue #1 缺陷清单"暖幢栋/抸扇/铺条官路/头一份"等直接初始化——
//     这些是已发生过的缺陷，默认就应该是黑名单。
//
// 各段与门禁的对应：
//
//	format        格式门（字数容差）
//	hook_payoff   钩子/回收门（悬置集数阈值）
//	hygiene       文本卫生门（错别字/乱码/生僻字表）
//	emotion       情绪曲线门（类型表 + 连续同类型上限）
//	arc           弧线门（主角等级轨 + 升速 + 代价）
//	producibility 可拍性门（角色/场景/群演/镜头语言/上屏汉字）
//	compliance    品牌安全门（分级词表）
//	novelty       新鲜度门（申报下限 + 重复度阈值）
//	voice         声音指纹门（指纹容差）

// Config 是 config.yaml 的根：各门禁一段，字段名与门禁 id 一致。
type Config struct {
	Format        FormatConfig        `yaml:"format"`
	HookPayoff    HookPayoffConfig    `yaml:"hook_payoff"`
	Hygiene       HygieneConfig       `yaml:"hygiene"`
	Emotion       EmotionConfig       `yaml:"emotion"`
	Arc           ArcConfig           `yaml:"arc"`
	Producibility ProducibilityConfig `yaml:"producibility"`
	Compliance    ComplianceConfig    `yaml:"compliance"`
	Novelty       NoveltyConfig       `yaml:"novelty"`
	Voice         VoiceConfig         `yaml:"voice"`
}

// FormatConfig 格式门：字数区间容差（target_chars ± tolerance）。
type FormatConfig struct {
	CharsTolerance float64 `yaml:"chars_tolerance"` // 如 0.10 = ±10%
}

// HookPayoffConfig 钩子/回收门：open loop 悬置阈值（issue #1 §B-2 门 7）。
type HookPayoffConfig struct {
	WarnAfterEps int `yaml:"warn_after_eps"` // 悬置 >6 集告警
	FailAfterEps int `yaml:"fail_after_eps"` // 悬置 >10 集硬失败
}

// HygieneConfig 文本卫生门（issue #1 §B-2 门 2）：词表即缺陷黑名单。
type HygieneConfig struct {
	Typos           map[string]string `yaml:"typos"`            // 错别字/异体字 → 修正建议（愣→怔）
	GarbledPatterns []string          `yaml:"garbled_patterns"` // 乱码/未成词片段（暖幢栋、快仗的通道）
	RareChars       []string          `yaml:"rare_chars"`       // 生僻字表（供 TTS 预读检查）
}

// EmotionConfig 情绪曲线门（软门 14）：类型表 + 连续同类型上限。
type EmotionConfig struct {
	Types     []string `yaml:"types"`      // 情绪类型全集（惊叹/共情/打脸/温暖/悬疑/危机）
	MaxStreak int      `yaml:"max_streak"` // 连续 N 集同类型即 fail
}

// ArcConfig 弧线门（issue #1 §B-2 门 9）：主角权限等级轨。
type ArcConfig struct {
	Character     string `yaml:"character"`        // 追踪的 entity id
	MaxLevel      int    `yaml:"max_level"`        // 等级上限（0→5）
	MinEpsPerStep int    `yaml:"min_eps_per_step"` // 每次升级至少间隔集数（每5集最多+1）
}

// ProducibilityConfig 可拍性门（issue #1 §B-2 门 12）。
type ProducibilityConfig struct {
	MaxNamedCharsPerEp int      `yaml:"max_named_chars_per_ep"` // 每集具名角色 ≤5
	MaxNewCharsPerEp   int      `yaml:"max_new_chars_per_ep"`   // 每集新角色 ≤1
	MaxScenesPerEp     int      `yaml:"max_scenes_per_ep"`      // 每集场景 ≤3
	MaxCrowdScenes     int      `yaml:"max_crowd_scenes"`       // 群演场次全剧配额上限
	CameraTerms        []string `yaml:"camera_terms"`           // 散文禁止的镜头语言（"镜头一抬"）
	OnscreenTriggers   []string `yaml:"onscreen_triggers"`      // 剧情关键汉字上屏触发词（刻着/匾额/纸条上）
}

// ComplianceConfig 品牌安全门（issue #1 §B-2 门 11）：分级词表。
type ComplianceConfig struct {
	Categories []ComplianceCategory `yaml:"categories"`
}

// ComplianceCategory 一类合规风险词表；Level 取值：
//
//	hard  硬失败（官商往来/特权/迷信类，真实品牌不能背）
//	flag  标记人审（绝对化用语等，进风险清单，剪宣传物料前必须过一遍）
type ComplianceCategory struct {
	ID       string   `yaml:"id"`
	Level    string   `yaml:"level"` // hard | flag
	Patterns []string `yaml:"patterns"`
}

// Compliance level 取值。
const (
	ComplianceHard = "hard"
	ComplianceFlag = "flag"
)

// NoveltyConfig 新鲜度门（issue #1 §B-2 门 8）。
type NoveltyConfig struct {
	MinNewFacts      int     `yaml:"min_new_facts"`      // 每集必须申报 ≥1 new_fact
	MinStateChanges  int     `yaml:"min_state_changes"`  // 每集必须申报 ≥1 state_change
	MaxRepeatRatio   float64 `yaml:"max_repeat_ratio"`   // 与前 N 集的 n-gram 重复率上限
	CompareWindowEps int     `yaml:"compare_window_eps"` // 相似度比较窗口（前 N 集）
}

// VoiceConfig 声音指纹门（软门 13）：台词指纹离散度下限——
// 所有角色句长/文白比几乎无差别 = "所有人说话一个味儿"。
type VoiceConfig struct {
	MinProfileSpread float64 `yaml:"min_profile_spread"` // 角色间平均句长标准差下限（字）
}

// WithDefaults 返回填充内置默认值后的配置副本：零值字段按 issue #1 的推荐值补齐。
// canon 目录未提供 config.yaml 时，门禁拿到的就是这份默认值。
func (c Config) WithDefaults() Config {
	out := c
	if out.Format.CharsTolerance <= 0 {
		out.Format.CharsTolerance = 0.10
	}
	if out.HookPayoff.WarnAfterEps <= 0 {
		out.HookPayoff.WarnAfterEps = 6
	}
	if out.HookPayoff.FailAfterEps <= 0 {
		out.HookPayoff.FailAfterEps = 10
	}
	if len(out.Hygiene.Typos) == 0 {
		out.Hygiene.Typos = map[string]string{
			"愣":    "怔",
			"抸扇":   "折扇",
			"针砭丝线": "针线",
			"抿嘴一按": "抿嘴一抿",
		}
	}
	if len(out.Hygiene.GarbledPatterns) == 0 {
		out.Hygiene.GarbledPatterns = []string{"暖幢栋", "快仗的通道", "月门外色地候着", "那可门影"}
	}
	if len(out.Emotion.Types) == 0 {
		out.Emotion.Types = []string{"惊叹", "共情", "打脸", "温暖", "悬疑", "危机"}
	}
	if out.Emotion.MaxStreak <= 0 {
		out.Emotion.MaxStreak = 3
	}
	if out.Arc.MaxLevel <= 0 {
		out.Arc.MaxLevel = 5
	}
	if out.Arc.MinEpsPerStep <= 0 {
		out.Arc.MinEpsPerStep = 5
	}
	if out.Producibility.MaxNamedCharsPerEp <= 0 {
		out.Producibility.MaxNamedCharsPerEp = 5
	}
	if out.Producibility.MaxNewCharsPerEp <= 0 {
		out.Producibility.MaxNewCharsPerEp = 1
	}
	if out.Producibility.MaxScenesPerEp <= 0 {
		out.Producibility.MaxScenesPerEp = 3
	}
	if out.Producibility.MaxCrowdScenes <= 0 {
		out.Producibility.MaxCrowdScenes = 2
	}
	if len(out.Producibility.CameraTerms) == 0 {
		out.Producibility.CameraTerms = []string{"镜头一抬", "镜头一转", "特写", "全景扫过", "推近", "拉远", "摇镜"}
	}
	if len(out.Producibility.OnscreenTriggers) == 0 {
		out.Producibility.OnscreenTriggers = []string{"刻着", "匾额", "招牌上", "纸条上", "写着", "屏风上绣", "荐状", "聘书"}
	}
	if len(out.Compliance.Categories) == 0 {
		out.Compliance.Categories = []ComplianceCategory{
			{ID: "official-favor", Level: ComplianceHard, Patterns: []string{"铺条官路", "包你吃香喝辣", "免费住一宿", "宿上一宿", "赏你几分"}},
			{ID: "privilege", Level: ComplianceHard, Patterns: []string{"有令牌才算有脸面", "有脸面"}},
			{ID: "superstition", Level: ComplianceHard, Patterns: []string{"妖火", "摄魂", "邪门勾当"}},
			{ID: "absolute", Level: ComplianceFlag, Patterns: []string{"头一份", "汴京第一", "人人争的"}},
		}
	}
	if out.Novelty.MinNewFacts <= 0 {
		out.Novelty.MinNewFacts = 1
	}
	if out.Novelty.MinStateChanges <= 0 {
		out.Novelty.MinStateChanges = 1
	}
	if out.Novelty.MaxRepeatRatio <= 0 {
		out.Novelty.MaxRepeatRatio = 0.6
	}
	if out.Novelty.CompareWindowEps <= 0 {
		out.Novelty.CompareWindowEps = 3
	}
	if out.Voice.MinProfileSpread <= 0 {
		out.Voice.MinProfileSpread = 4.0
	}
	return out
}
