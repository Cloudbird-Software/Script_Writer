# ADR-0001：语言切换为 Go，本仓定位 songguard（跨集一致性校验工具）

- 状态：accepted · 日期：2026-08-19 · 决策来源：issue #1 及其批准评论（owner 全部批准）

## 背景

issue #1 对三十集生成小说的缺陷复盘结论：约 90% 缺陷是**跨集缺陷**，根因是缺
"世界状态机（canon snapshot/delta/open-loop ledger）+ 全局校验 pass"，继续加严单集门禁无效。
需从 0 建设跨集校验工具。

## 决定

1. **语言切换为 Go**。依据组织 `governance/policy/languages.yaml`：
   - application 层 default = go；本工具是纯后端校验 + CLI，无前端；
   - `language_change.rule`：语言选定后更换 = 重新立项——趁仓库尚无业务代码现在切换成本≈0；
   - M1~M3 规则校验核心零 LLM 依赖，恰是纯 Go 最稳的部分；GO-1~GO-5 现成。
2. **本仓定位 songguard**：纯函数校验工具集（Go 库 + thin CLI），**不含 LLM 生成编排**。
   M5 的 LLM 软能力（Novelty embedding / Reader Simulation / Sweep 审校）届期按
   llm_prompt 层政策独立成 python+baml 旁路仓，Go CLI 经子进程/HTTP 调用。
3. **分期 M1~M3**（每期可独立交付，按"拦 P0 能力"排序）：
   - M1 状态内核：六张 canon 表（YAML）schema + delta 协议 + apply + 伏笔/相遇/道具台账 + 时间轴；
   - M2 五道硬门：一致性 / 关系 / 引文接地 / 钩子回收 / 格式；
   - M3 全局 pass：Ledger Close（五件套报表）+ Consistency Sweep 规则版（只出 diff 建议）+ ±1 集联动校验。
   - M4/M5（域门禁纯规则版、LLM 软能力）后置，不在本期。
4. **已批新依赖**（dependency_policy 报批记录）：
   - `gopkg.in/yaml.v3`（Apache-2.0，canon 表解析，标准库无 YAML）
   - `github.com/flyingmutant/rapid`（MIT，PBT，verification 层强制）
   - errcheck 未批，lint 暂为 gofmt+go vet，报批后补（GO-2 完整落地）。

## 影响

- 移除 Python NSC 引擎与其监控 CI（metrics/llm-eval/judge-calibration/rule-promotion）：
  其 LLM 编排职责与 songguard 定位不符；代码保留在 git 历史（PR #3）与原个人仓，
  M5 旁路仓需要时按 llm_prompt 层政策重启。
- CI `check` runtime 切 go；CodeQL default-setup 切 go；dependabot 切 gomod。
- Makefile 目标名不变（setup/fmt/lint/arch/test/build/check），实现切 Go 工具链。

## 后果与验证

- 每道门禁必须带 issue #1 实际缺陷的复现用例（E5 渔捕快、E9 A福、E3 靖康驿站、
  E12 二次相识、E30 过客有期、E14→E15 钩子断裂、E21 字数）。
- 不变量用 PBT：台账守恒（closed ⊆ opened）、时间轴单调、合法序列零违规。
