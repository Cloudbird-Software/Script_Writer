# 架构纪律（每个包都必须遵守）

> 新建包、动包边界、review 时读。组织政策：`governance/policy/languages.yaml`（application 层 = go）。

## 深接口总纲

**对外暴露的只有一个**：`internal/songguard` 门面包。`cmd/` 与任何未来调用方只 import 它；
其余 internal 包都是门面背后的分层内幕，越深越好，但不许再开第二个对外面。

## 分层与依赖方向（只允许单向）

```
cmd/songguard ─▶ internal/songguard ─▶ internal/engine ─▶ internal/rules ─▶ internal/state ─▶ internal/canon
      │                                     │                (rules/gates … 各门子包)
      └（cmd 只 import songguard；engine 只被 songguard 调；rules 子包只依赖 rules/state/canon；
        禁止反向/循环。`make arch` 执法 main 包只在 cmd/、internal 不 import cmd。）
```

- **internal/songguard（门面）**：全工具唯一对外接口——`New()` 构造、`Check()` 全量校验、
  `Linkage()` ±1 集联动校验；`Report`/`LinkageReport` 集中渲染与统计方法。
- **internal/engine（编排）**：manifest 加载、门禁注册表（registry）、Ledger Close 结算、
  Sweep 巡检、Linkage 联动。新增门禁只改 `registry.go` 一行 + 对应 rules 子包。
- **internal/rules（门禁）**：`Rule` 深契约（ID + Check 两个面）+ 共享文本工具。
  M2 五门在 `rules/gates`；M4 每门独立子包 `rules/<gate>`，子包只导出一个 `Rule()` 构造器。
- **internal/canon（M1+M4）**：六张 canon 表（Entity Registry / Prop Bible / World Rules /
  Line Assets / 卖点排期 / Timeline）+ 可选第七张 `config.yaml`（M4 门禁阈值与词表，
  缺省走 `WithDefaults()`，见 ADR-0002）的 Go 类型、YAML 加载与结构校验。
  不可变数据定义，不依赖任何其它 internal 包。
- **internal/state（M1）**：delta 协议（meetings / hooks / prop_changes / line_uses /
  selling_point / emotion / time / new_facts / state_changes / arc / scenes）+ apply +
  三本台账（伏笔 open-loop、相遇、道具 instance）+ 时间轴。纯函数，无 IO。
- **sidecar/（M5，Python 进程）**：LLM 旁路。BAML（`baml_src/*.baml`）封装 prompt
  ——prompt 是 harness 资产，git 即版本管理；对外唯一 HTTP 端点 `POST /v1/llm-check`
  （深接口的进程间形态），Go 侧经 `internal/llm` 客户端调用。LLM 结论一律建议级，
  阻断决策留在 Go 规则门禁（ADR-0003）。

## 包纪律

1. **GO-3**：入口只在 `cmd/`，`package main` 不得出现在 internal/；internal 不得 import cmd。
2. **对外行为契约** = `songguard` 门面导出 API + 各包注释；跨包禁止深挖内部实现。
3. **包 ≤3000 行**（组织 MOD-3）。超限就拆——一个包必须能被 agent 一次性完整读完。
4. **接口设计标准（IF-1）**：一个 LLM 能否仅凭函数签名 + 一行注释零样本正确使用？否 => 重做。
5. **测试纪律**：
   - 每道门禁必须带 issue #1 实际缺陷的**复现用例**（E5 渔捕快、E9 A福、E3 靖康驿站、
     E12 二次相识、E30 过客有期、E14→E15 钩子断裂、E21 字数）。
   - 行为不变量用 PBT（`flyingmutant/rapid`）：apply 台账守恒（closed ⊆ opened）、
     时间轴单调、合法序列零违规。
   - bug 修复先写复现失败测试（AGENTS.md 规则 5）。

## 数据规则

- canon 表与 delta 申报是**唯一事实来源**；正文文件只读，任何 pass 不得改写正文
  （Sweep 只输出 diff 建议）。
- 六张表 YAML schema 变更 = 资产层变更：打 `asset-change` 标签 + 附 ADR。
