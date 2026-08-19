# 架构纪律（每个包都必须遵守）

> 新建包、动包边界、review 时读。组织政策：`governance/policy/languages.yaml`（application 层 = go）。

## 分层与依赖方向（只允许单向）

```
cmd/songguard ──▶ internal/passes ──▶ internal/gates ──▶ internal/state ──▶ internal/canon
      │                                                                              ▲
      └──────────────────────────────────────────────────────────────────────────────┘
（cmd 可读任意 internal 包；internal 包只能向左依赖，禁止反向/循环。`make arch` 执法。）
```

- **internal/canon**（M1）：六张 canon 表（Entity Registry / Prop Bible / World Rules /
  Line Assets / 卖点排期 / Timeline）的 Go 类型、YAML 加载与结构校验。
  不可变数据定义，不依赖任何其它 internal 包。
- **internal/state**（M1）：delta 协议（meetings / hooks / prop_changes / line_uses /
  selling_point / emotion / time / new_facts / state_changes）+ apply + 三本台账
  （伏笔 open-loop、相遇、道具 instance）+ 时间轴。纯函数，无 IO。
- **internal/gates**（M2）：五道硬门——一致性 / 关系 / 引文接地 / 钩子回收 / 格式。
  输入 canon + 台账 + 正文，输出 `[]Violation`。纯函数，零 LLM。
- **internal/passes**（M3）：Ledger Close（交付五件套报表）、Consistency Sweep 规则版
  （**只输出 diff 建议、不许重写全文**）、±1 集联动校验。

## 包纪律

1. **GO-3**：入口只在 `cmd/`，`package main` 不得出现在 internal/；internal 不得 import cmd。
2. **对外行为契约** = 各包导出函数签名 + 包注释；跨包禁止深挖内部实现。
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
