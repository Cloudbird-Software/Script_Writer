// Package songguard 是全工具的唯一对外门面（深接口）。
//
// 调用方（cmd/ 与未来嵌入方）只 import 本包：
//
//	g := songguard.New()                 // 门面唯一构造入口
//	rep, err := g.Check("manifest.yaml") // 全量校验
//	out, err := g.Linkage("m.yaml", 4)   // 重跑 ±1 集联动校验
//
// 内部分层（依赖只允许单向，见 docs/ARCHITECTURE.md）：
//
//	songguard（门面）─▶ engine（编排）─▶ rules（门禁）─▶ state（台账）─▶ canon（六表）
//
// Report/LinkageReport 把渲染与统计方法集中到门面，调用方无需接触任何内层包。
package songguard

import (
	"fmt"

	"github.com/Cloudbird-Software/Script_Writer/internal/engine"
)

// Guard 是跨集一致性校验的执行器；零值不可用，须 New 构造。
type Guard struct {
	opts options
}

// New 创建 Guard；可选配置项见 Option。
func New(opts ...Option) *Guard {
	o := defaultOptions()
	for _, fn := range opts {
		fn(&o)
	}
	return &Guard{opts: o}
}

// Check 对 manifest 做全量校验：canon 结构 → 状态台账 → 全部门禁 → 全局 pass。
func (g *Guard) Check(manifestPath string) (*Report, error) {
	c, eps, err := engine.LoadManifest(manifestPath)
	if err != nil {
		return nil, err
	}
	res, err := engine.Run(c, eps)
	if err != nil {
		return nil, err
	}
	return &Report{res: res}, nil
}

// Linkage 重跑 ±1 集联动校验：被重跑集与其前后集的钩子承接完整性
// （issue #1 §D-2，拦 E14→E15 类重跑断裂）。
func (g *Guard) Linkage(manifestPath string, ep int) (*LinkageReport, error) {
	if ep <= 0 {
		return nil, fmt.Errorf("songguard: 集号必须 ≥1，得到 %d", ep)
	}
	c, eps, err := engine.LoadManifest(manifestPath)
	if err != nil {
		return nil, err
	}
	for _, e := range eps {
		if e.Ep == ep {
			return &LinkageReport{Ep: ep, vs: engine.Linkage(c, eps, ep)}, nil
		}
	}
	return nil, fmt.Errorf("songguard: E%d 不在 manifest episodes 中", ep)
}
