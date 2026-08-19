# ADR-0003：M5 LLM 旁路采用 Python+BAML sidecar，进程外 HTTP 门面

日期：2026-08-20
状态：已接受

## 背景

issue #1 的方案里有两类检查：规则可判的（十二道门+两道软门的主体）与语义可判的
（Pass 1 一致性巡检的"近似漂移"、Pass 3 观众模拟的钩子强度/弃剧点/令牌复述测试、
文白比/口头禅指纹、代词消解）。前者已在 Go 内核落地；后者必须调 LLM，且 prompt
本身是 harness 的一部分——**prompt 会迭代，迭代必须可回归**。

## 决策

1. **LLM 能力放 sidecar 进程**（`sidecar/`，Python），Go 主进程通过 HTTP 调用。
   不在 Go 内嵌 prompt 字符串、不引 Go 版 LLM SDK。
2. **prompt 用 BAML 封装**（`sidecar/baml_src/*.baml`）：函数签名（入参/输出
   class）与 prompt 模板同文件声明；git 即 prompt 的版本管理；生成的 pydantic
   类型直接成为 sidecar 内部模型。
3. **对外只暴露一个 HTTP 端点** `POST /v1/llm-check`（深接口纪律的延续）：
   Go 侧 `internal/llm` 只依赖这一个契约，sidecar 内部层次（passes → baml_client
   → providers/mocksrv）对调用方不可见。
4. **LLM 结论一律建议级（warn）**：阻断决策永远在 Go 侧规则门禁——旁路挂了
   最多少一层建议，交付主流程不受影响。
5. **mock 优先**：默认 provider=mock，内置 mock LLM（OpenAI 兼容），走完整
   BAML 渲染/解析管线（不绕过 BAML 直接造对象）；无 API key 也能全量测试。
   `sidecar/fixtures/llm_contract.json` 作为 Go/Python 双侧共享契约期望。

## 备选方案与不选的原因

- **Go 内嵌 prompt + 直接 HTTP 调 LLM**：prompt 失去结构化 schema 约束，
  解析鲁棒性差（BAML 的 SAP 解析正是其强项），且 prompt 迭代无 diff 可审。
- **进程内 cgo/FFI 调 BAML**：BAML 无稳定 Go 运行时（仅 python/ts/ruby），
  cgo 会把部署复杂度转嫁给 CI。
- **LangChain/pydantic-ai 等 Python 框架**：依赖重；我们的需求（模板+schema+解析）
  BAML 一个文件解决，AGENTS.md 依赖审批最小化。

## 影响

- 新增运行时依赖（sidecar 进程）：`baml-py`、`pydantic`（均 MIT，用户已在任务中
  点名使用 BAML）。Go 侧零新依赖（标准库 net/http）。
- 改 prompt 流程：改 `.baml` → `generate.sh` → 提交生成产物（CI 不需要 baml-cli）。
- 真实 LLM 走任何 OpenAI 兼容端点（env 配置，密钥不落仓库）。
