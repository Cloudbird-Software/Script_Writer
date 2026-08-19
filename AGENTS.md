# AGENTS.md（索引型——只放不可推断的约束，细节按需读索引）

## 命令

- `make setup` 安装 / `make check` 提交前必跑（lint+arch+test）/ `go test ./internal/<pkg>/ -run <Test>` 单测

## 硬规则（违反 = PR 打回）

1. 认证：一切 push/PR 用 cloudbrid-agent App 令牌，禁个人 PAT。获取：
   `GH_TOKEN=$(REPO=Script_Writer bash <(curl -sS https://raw.githubusercontent.com/Cloudbird-Software/.github/main/scripts/gh-app-token.sh))`
2. 不改 `.github/workflows/**`、`Makefile` 的 check 目标（App 无此权限，人类专属）
3. 新依赖先报"名称/用途/许可证/标准库可否替代"等人批；禁 AGPL/GPL-3.0/SSPL
4. 密钥、客户名、连接串不进仓库，用 `.env.example` 占位
5. 一个 PR 一件事，diff < 400 行；bug 修复先写复现失败测试
6. 对外接口变更写 CHANGELOG.md；提交信息用 Conventional Commits

## 索引（用到再读，不要全读）

| 场景 | 读这个 |
| --- | --- |
| 改 canon 表 schema / 六张表结构 | [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) + `internal/canon/` |
| 改 delta 协议 / 台账（伏笔/相遇/道具/时间轴） | `internal/state/` 包注释 |
| 加/改门禁（五道硬门） | `internal/gates/` 包注释 + [issue #1](https://github.com/Cloudbird-Software/Script_Writer/issues/1) §B-2 |
| 改全局 pass / 报表 / CLI | `internal/passes/` + `cmd/songguard/` |
| 决策记录（为什么是 Go / M1~M3 分期） | [adr/](adr/) |
| 选语言 / 选库 / 测试政策 | 组织 [governance/policy](https://github.com/Cloudbird-Software/.github/tree/main/governance/policy) |
