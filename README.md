# Script_Writer · songguard

小说/短剧**跨集一致性校验工具**（Go 库 + thin CLI）。名字来自它的职责：给"歌"站岗——
三十集连续生成时，把单集门禁拦不住的**跨集缺陷**（人名漂移、伏笔失联、引文无出处、钩子断裂……）在交付前拦下。

- 缘起与完整缺陷清单：[issue #1](https://github.com/Cloudbird-Software/Script_Writer/issues/1)
- 决策记录：[adr/0001](adr/0001-lang-switch-to-go-songguard.md)（Python→Go 语言切换 + songguard 定位）
- 语言定位：组织 `languages.yaml` **application 层 default = go**（GO-1~GO-5 现成）

## 它校验什么

```
输入：六张 canon 表(YAML) + 各集正文 + 各集 delta 申报
输出：violation[]{gate, episode, 位置, expected, actual, severity}
      + 交付五件套报表（人物表 / 伏笔台账 / 卖点覆盖表 / 风险清单 / 每集 beat+钩子表）
```

| 层 | 内容 | 状态 |
| --- | --- | --- |
| M1 状态内核 | 六张 canon 表 schema + delta 协议 + apply + 伏笔/相遇/道具台账 + 时间轴 | 开发中 |
| M2 五道硬门 | 一致性 / 关系 / 引文接地 / 钩子回收 / 格式（纯函数，零 LLM） | 开发中 |
| M3 全局 pass | Ledger Close 结算 + Consistency Sweep（只出 diff 建议）+ ±1 集联动校验 | 开发中 |
| M4/M5 | 域门禁纯规则版 / LLM 软能力（届时按 llm_prompt 层政策独立成 python+baml 旁路仓） | 后置 |

## 快速开始

```bash
make setup                                # go mod download
make check                                # gofmt + go vet + 边界检查 + go test -race
go run ./cmd/songguard check manifest.yaml # M3 交付
```

## Makefile 接口（所有语言统一，CI 只认这个）

| 目标 | 作用 |
| --- | --- |
| `make setup` | 安装依赖（go mod download） |
| `make fmt` | gofmt 格式化 |
| `make lint` | gofmt --check + go vet（errcheck 待依赖报批后加入，GO-2） |
| `make arch` | GO-3 边界检查：main 包只允许在 cmd/、internal 不得 import cmd |
| `make test` | go test -race ./...（GO-4） |
| `make build` | 构建 bin/songguard |
| `make check` | lint + arch + test，**提交前必须全绿** |

## CI 结构

- `hygiene`：密钥扫描（gitleaks 全历史）、大文件/凭据文件拦截、zizmor Actions 审计
- `check`：`make setup && make check`（runtime: go）
- `deps`：依赖漏洞 + 许可证审查（PR 时）
- `gate`：聚合门（组织 ruleset 的唯一必需 check）

基础工作流实现在 [CI-Workflows](https://github.com/Cloudbird-Software/CI-Workflows)，本仓只引用 `@v1`。

## 目录

```
cmd/songguard/     CLI 入口（GO-3）
internal/canon/    M1: 六张 canon 表的类型 + 加载 + 结构校验
internal/state/    M1: delta 协议 + apply + 伏笔/相遇/道具台账 + 时间轴
internal/gates/    M2: 五道硬门（纯函数）
internal/passes/   M3: Ledger Close + Sweep 规则版 + ±1 集联动校验
```

架构纪律见 [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)；开发规矩见 [AGENTS.md](AGENTS.md)。
