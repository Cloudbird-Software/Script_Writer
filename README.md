# Script_Writer（NSC · Narrative Spec Compiler）

把「写小说 / 写剧本」变成**编译**：`spec/` 是源码，小说、剧本、prompt、代码都是编译产物。

- 业务：为商家生成可直接发在自有短视频账号的**营销短剧**。先出**小说**（商家确认物），再出**剧本**（制作团队执行物）。
- 视频拍摄/剪辑由外部制作团队负责，本系统**不做视频生成**。
- 语言定位：组织 `languages.yaml` 的 **llm_prompt 层**（Python 允许域）。

## 快速开始

```bash
make setup        # uv sync --all-extras
make db-rebuild   # 从 cases/export/*.jsonl 重建 SQLite 工作副本
make check        # lint + typecheck + 资产守卫 + 测试（无 LLM，提交前必绿）
nsc run --brief examples/demo_tea/brief.yaml --profile short_drama_v1
nsc check out/demo_tea/ir.json
nsc render out/demo_tea/ir.json --target novel_docx script_fountain
```

## Makefile 接口（所有语言统一，CI 只认这个）

| 目标 | 作用 |
| --- | --- |
| `make setup` | 安装依赖（uv sync） |
| `make fmt` | ruff 格式化 + 修复 |
| `make lint` | ruff format --check + ruff check + pyright |
| `make arch` | 资产层完整性守卫（7 个 spec-guard，等价 depcruise 位） |
| `make test` | db-rebuild + pytest（not llm）+ coverage ≥80 |
| `make build` | uv build |
| `make check` | lint + arch + test，**提交前必须全绿** |

## CI 结构

- `hygiene`：密钥扫描（gitleaks）、大文件/凭据文件拦截、zizmor Actions 审计
- `check`：`make setup && make check`（python 3.12 / uv）
- `deps`：依赖漏洞 + 许可证审查（PR 时）
- `gate`：聚合门（组织 ruleset 的唯一必需 check）
- 监控 CI（本仓特有）：`metrics.yml` 周报 + 北极星告警、`llm-eval.yml` L1 判官评测、
  `judge-calibration.yml` 判官校准关/开闸、`rule-promotion.yml` 规则挖掘晋升 PR

基础工作流实现在 [CI-Workflows](https://github.com/Cloudbird-Software/CI-Workflows)，本仓只引用 `@v1`。

## 三条铁律

1. **`spec/` 是唯一真相。** 任何知识若不能落进 `spec/ | profiles | brands`，视为不存在。
2. **`src/`、`out/` 是生成物层。** 重写 `src/` 必须能通过同一套 `tests/`。
3. **反馈必须锚定到节点 ID。** 无 `node_id` 的反馈不入库；案例真相在 `cases/export/*.jsonl`。

详见 [AGENTS.md](AGENTS.md)、[src/AGENTS.md](src/AGENTS.md)、[docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)、[COMPLIANCE.md](COMPLIANCE.md)。
