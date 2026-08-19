# Changelog

本文件记录对外可见的变更。格式参考 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/)。

## [Unreleased]

### Added
- 初始模板工程（CI gate / hygiene / dependabot / automerge 全套护栏）。
- 迁入 NSC（Narrative Spec Compiler）：spec 资产层、生成引擎（src/nsc）、
  测试与 fixtures、ADR、飞轮数据真相（cases/export/*.jsonl）、监控 CI
  （metrics / llm-eval / judge-calibration / rule-promotion）。
- Makefile 对齐组织统一接口（setup/fmt/lint/arch/test/build/check，python + uv）。

### Changed
- 监控 CI 对齐组织治理：BP-1（周报/校准不再直推 main，走 artifact/Issue/PR）、
  CI-2/CI-3（Actions 全 SHA pin、最小权限、persist-credentials: false）、
  lint+test 收敛进组织 check.yml 的 `make check`。

### Removed
- 迁移过程文件：Engineering-Plan、docs/WORK_ORDERS.md、docs/UPGRADE_PLAN_*、
  docs/RESEARCH_EXTERNAL_*、历史周报 docs/metrics/*、cases/cases.db（工作副本）、
  .pre-commit-config.yaml、旧 rulesets/main.json、Node 模板残留（package.json 等）。
