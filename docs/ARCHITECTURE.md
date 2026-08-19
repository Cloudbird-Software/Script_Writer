# 架构纪律（每个模块都必须遵守）

> 新建模块、动模块边界、review 时读。开发循环的硬约束见 [src/AGENTS.md](../src/AGENTS.md)。

## 分层

```
spec/（资产 A：ir / checks / rubrics / rules / passes 契约）   ← 唯一真相，改它要 ADR
src/nsc/（代码 B2：runtime·checker·passes·judge·optimize …）   ← 可丢弃，测试全绿即可重写
cases/export/*.jsonl（资产 A：飞轮数据真相）→ cases.db（B3 工作副本，gitignored）
out/（B3 生成物，gitignored）
```

## 模块纪律

1. **编译 Pass 的 public entry 是 `spec/passes/signatures.py` 的签名契约**；Pass 间依赖必须与
   `spec/passes/dep_graph.yaml` 一致。跨模块不得绕过契约直接 import 内部函数。
2. **`src/nsc/guards/` 是本仓的架构边界 lint（`make arch`）**：D2 归约、规则 schema、
   prompts 未手改、IR breaking change、行数预算（`spec/BUDGETS.yaml`）、规则冲突、
   db↔jsonl 一致。对应组织模板的 depcruise 位。
3. **模块大小上限 3000 行**（`spec/BUDGETS.yaml` 按 D21 预算执法）。超过就拆——
   一个模块必须能被 agent 一次性完整读完。
4. **生成代码独立目录，禁手改**（`out/`、渲染产物；`prompts/` 只能由 `nsc optimize` 写入）。
5. **接口设计标准**：一个 LLM 能否仅凭函数签名 + 一行 docstring 就零样本正确使用？
   答案是否 => 接口太浅，重做。
6. **测试优先级**：行为不变量用 property-based test（pytest + hypothesis），
   关键输出用 golden test（syrupy 快照 + `tests/fixtures/golden/`）。
   先写不变量，再写实现。每条 check 规则必须带 `pass.json` / `fail.json` fixture。

## 依赖规则

新增依赖前先列出"依赖名 / 用途 / 许可证 / 是否能用标准库替代"等人确认；
禁止引入 AGPL / GPL-3.0 / SSPL 的库（组织 `policy/languages.yaml`）。
LLM 调用一律经 `src/nsc/runtime/models.py` 路由（模型 alias 声明在 `config/models.yaml`）。

## 数据规则

- 唯一数据库 = SQLite + sqlite-vec；持久形态 = `cases/export/*.jsonl`（ADR-0006）。
- 禁止重 ORM（组织 `policy/languages.yaml` data 层）。
