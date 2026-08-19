# AGENTS.md（索引型——只放不可推断的约束，细节按需读索引）

## 命令

- `make setup` 安装 / `make check` 提交前必跑（lint+arch+test）/ `make test <文件>` 单测

## 硬规则（违反 = PR 打回）

1. 认证：一切 push/PR 用 cloudbrid-agent App 令牌，禁个人 PAT。获取：
   `GH_TOKEN=$(REPO=template-service bash <(curl -sS https://raw.githubusercontent.com/Cloudbird-Software/.github/main/scripts/gh-app-token.sh))`
2. 不改 `.github/workflows/**`、`Makefile` 的 check 目标（App 无此权限，人类专属）
3. 新依赖先报"名称/用途/许可证/标准库可否替代"等人批；禁 AGPL/GPL-3.0/SSPL
4. 密钥、客户名、连接串不进仓库，用 `.env.example` 占位
5. 一个 PR 一件事，diff < 400 行；bug 修复先写复现失败测试
6. 对外接口变更写 CHANGELOG.md；提交信息用 Conventional Commits

## 索引（用到再读，不要全读）

| 场景 | 读这个 |
| --- | --- |
| 改生成 Pass / 模块边界 / 层级规矩 | [src/AGENTS.md](src/AGENTS.md) + [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) |
| 改检查规则 | `spec/checks/DSL.md` → `spec/checks/_schema.yaml` |
| 改判官 / 评测 | `spec/rubrics/` → [docs/SOP_JUDGE_CALIBRATION.md](docs/SOP_JUDGE_CALIBRATION.md) |
| 反馈 / 飞轮数据 | [docs/SOP_FEEDBACK_INGEST.md](docs/SOP_FEEDBACK_INGEST.md)（真相 = `cases/export/*.jsonl`） |
| 决策记录 / 为什么长这样 | [adr/](adr/) |
| 选语言 / 选库 / 测试政策 | 组织 [governance/policy](https://github.com/Cloudbird-Software/.github/tree/main/governance/policy) |
