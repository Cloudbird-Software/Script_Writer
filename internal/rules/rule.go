// Package rules 是门禁层（issue #1 §B-2 十二道硬门 + 两道软门）的深契约与共享工具。
//
// 深接口纪律：每道门禁对外只暴露一个入口——
//
//	存量五门（M2）：rules/gates 包的纯函数（Format/Consistency/…）经 engine 注册表适配
//	新增各门（M4）：每门一个子包 rules/<gate>，子包只导出一个 Rule() 构造器
//
// 门禁只读输入（Input 快照），输出 []state.Violation；不改写正文、不动台账。
package rules

import (
	"github.com/Cloudbird-Software/Script_Writer/internal/canon"
	"github.com/Cloudbird-Software/Script_Writer/internal/state"
)

// Input 是一道门禁的只读输入快照（canon + 全部集 + 各集后状态快照）。
type Input struct {
	Canon     *canon.Canon
	Episodes  []state.Episode
	Snapshots []state.Snapshot // 与 Episodes 等长同序；无台账输入的门禁可忽略
}

// Rule 是门禁的深契约：实现方只暴露 ID 与 Check 两个面。
type Rule interface {
	// ID 返回门禁标识（与 state.Gate* 常量一致），进违规报告的 gate 字段。
	ID() string
	// Check 对全量输入执行校验，返回违规（可为空）。
	Check(in Input) []state.Violation
}

// FuncRule 把签名形如 (canon, eps, snaps) 的纯函数适配为 Rule（存量 M2 五门用）。
type FuncRule struct {
	Gate string
	Fn   func(c *canon.Canon, eps []state.Episode, snaps []state.Snapshot) []state.Violation
}

// ID 实现 Rule。
func (r FuncRule) ID() string { return r.Gate }

// Check 实现 Rule。
func (r FuncRule) Check(in Input) []state.Violation {
	return r.Fn(in.Canon, in.Episodes, in.Snapshots)
}
