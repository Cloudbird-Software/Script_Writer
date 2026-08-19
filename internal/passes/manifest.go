// Package passes 实现 M3 全局 pass（issue #1 §C/D）：
//
//	Manifest    加载 manifest.yaml（canon 目录 + 各集正文 + delta 申报）
//	Run         全量编排：canon 结构校验 → 状态台账 apply → 五道门禁 → 结算
//	LedgerClose 交付五件套报表（人物表/伏笔台账/卖点覆盖表/风险清单/每集 beat+钩子表）
//	Sweep       一致性巡检规则版：只输出 diff 建议，绝不重写全文
//	Linkage     重跑 ±1 集联动校验（E14→E15 类重跑断裂）
package passes

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/Cloudbird-Software/Script_Writer/internal/canon"
	"github.com/Cloudbird-Software/Script_Writer/internal/state"
)

// manifestFile 是 manifest 的磁盘形态；episode 正文按 file 相对 manifest 所在目录读取。
type manifestFile struct {
	Canon    string            `yaml:"canon"`
	Episodes []manifestEpisode `yaml:"episodes"`
}

type manifestEpisode struct {
	Ep          int         `yaml:"ep"`
	File        string      `yaml:"file"`
	TargetChars int         `yaml:"target_chars"`
	Finale      bool        `yaml:"finale"`
	Delta       state.Delta `yaml:"delta"`
}

// LoadManifest 读取 manifest 并装载 canon 与全部正文，按集号升序返回。
func LoadManifest(path string) (*canon.Canon, []state.Episode, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("manifest: %w", err)
	}
	var m manifestFile
	if err := yaml.Unmarshal(raw, &m); err != nil {
		return nil, nil, fmt.Errorf("manifest: 解析失败: %w", err)
	}
	if m.Canon == "" {
		return nil, nil, fmt.Errorf("manifest: 缺少 canon 字段（canon 表目录）")
	}
	base := filepath.Dir(path)
	c, err := canon.Load(filepath.Join(base, m.Canon))
	if err != nil {
		return nil, nil, err
	}
	var eps []state.Episode
	for _, me := range m.Episodes {
		text, err := os.ReadFile(filepath.Join(base, me.File))
		if err != nil {
			return nil, nil, fmt.Errorf("E%d 正文缺失: %w", me.Ep, err)
		}
		eps = append(eps, state.Episode{
			Ep:          me.Ep,
			Text:        string(text),
			TargetChars: me.TargetChars,
			Finale:      me.Finale,
			Delta:       me.Delta,
		})
	}
	if len(eps) == 0 {
		return nil, nil, fmt.Errorf("manifest: episodes 为空")
	}
	// 按集号升序（台账与门禁都按序消费）。
	for i := 1; i < len(eps); i++ {
		for j := i; j > 0 && eps[j-1].Ep > eps[j].Ep; j-- {
			eps[j-1], eps[j] = eps[j], eps[j-1]
		}
	}
	return c, eps, nil
}
