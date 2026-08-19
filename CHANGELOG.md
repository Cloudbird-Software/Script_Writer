# Changelog

本文件记录对外可见的变更。格式参考 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/)。

## [Unreleased]

### Added
- M4 情绪曲线门 `emotion`（软门 14）：每集必须申报情绪类型且在类型表内，
  连续 N 集同类型即硬失败（打散"30 集一个套路"）。
- M4 文本卫生门 `hygiene`（门 2）：错别字/乱码片段/生僻字三张词表逐句扫描
  （愣→怔、暖幢栋、快仗的通道 等默认黑名单直接取材 issue #1 已发生缺陷）。
- M4 台词归属门 `lineownership`（门 6）：slogan/口头禅塞错嘴（句中唯一具名角色
  ≠ owner 即拦，E8/E19/E25/E28 类）+ 台词用量申报与正文一致性（未申报/虚报均拦；
  归属歧义句留给 M5 LLM 代词消解）。
- M4 资产层（ADR-0002）：canon 第七张表 `config.yaml`（可选）——九道门禁的阈值与词表
  （格式容差 / 钩子悬置阈值 / 错别字与乱码表 / 情绪类型 / 弧线等级轨 / 可拍性配额与镜头语言 /
  合规分级词表 / 新鲜度阈值 / 声音指纹容差），缺省走 `WithDefaults()`（默认词表直接取材
  issue #1 已发生缺陷）；delta 协议新增 `arc{level,cost}` / `scenes` / `crowd` 申报，
  台账新增主角等级轨（单调不倒退、单步 ≤+1）与群演累计。
- 初始模板工程（CI gate / hygiene / dependabot / automerge 全套护栏）。
- songguard 项目起架：Go 1.25 模块、`cmd/songguard` CLI 骨架、Makefile Go 工具链
  （lint=gofmt+go vet、arch=GO-3 边界检查、test=-race）、ADR-0001（语言切换与分期）。
- M1 状态内核（PR #5/#6）：六张 canon 表 schema/加载/结构校验（internal/canon）+ delta 协议/apply 引擎/伏笔·相遇·道具三本台账与时间轴（internal/state，PBT：closed⊆opened、apply 单调性）。
- M2 五道硬门（PR #7，internal/gates）：格式 / 一致性 / 关系 / 引文接地 / 钩子回收，全部带 issue #1 实际缺陷复现用例（E5"渔捕快"、E9"A福"、E12 二次相识、E30 检索失败、E21 字数）与 PBT。
- M3 全局 pass + CLI（internal/passes + cmd/songguard）：
  `songguard check [-out <dir>] <manifest.yaml>`（stdout 摘要 JSON + deliverable.md 交付五件套 + sweep.md 巡检建议 + violations.json）、
  `songguard linkage -manifest <m> -ep <N>`（重跑 ±1 集联动校验，拦 E14→E15 类断裂）；
  examples/demo 提供可跑通的干净样例（5 集）。

### Changed
- **语言切换 Python→Go**（ADR-0001，issue #1 批准）：仓库重定位为 songguard
  跨集一致性校验工具（纯函数，零 LLM）；CI `check` runtime→go、dependabot→gomod、
  README/AGENTS/ARCHITECTURE/CODEOWNERS/PR 模板按新结构重写。
- Makefile 目标名保持组织统一接口，实现切 Go。

### Removed
- Python NSC 引擎及全部配套（src/tests/spec/adr 旧表/cases/监控 CI ×4 等 467 文件）：
  LLM 编排职责与本仓新定位不符，保留于 git 历史（PR #3），M5 届期独立旁路仓重启。
