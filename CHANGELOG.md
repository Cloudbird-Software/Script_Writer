# Changelog

本文件记录对外可见的变更。格式参考 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/)。

## [Unreleased]

### Added
- M5 LLM 旁路 sidecar（ADR-0003）：`sidecar/` Python 进程，BAML 封装 prompt
  （`baml_src/*.baml` = prompt SSOT，git 即版本管理）+ 唯一 HTTP 端点
  `POST /v1/llm-check`（Pass 1 一致性巡检 sweep / Pass 3 观众模拟 reader）。
  默认 provider=mock（内置 OpenAI 兼容 mock LLM，走完整 BAML 渲染/解析管线），
  真实 LLM 走任意 OpenAI 兼容端点（env 配置）。`fixtures/llm_contract.json`
  为 Go/Python 双侧共享契约期望。LLM 结论一律建议级，阻断决策留在 Go 规则门禁。
- M4 声音指纹门 `voice`（软门 13，规则版）：引号台词按规则版归属（说话语境中
  唯一具名角色）分桶，各角色平均台词长度的标准差 < min_profile_spread 即 warn
  （"所有人说话一个味儿"；文白比/口头禅深查与代词消解属 M5 LLM 旁路）。
  台词归属门与声音指纹门共用 `rules.SoleCharacter` 归属判定（消除重复实现）。
- M4 新鲜度门 `novelty`（门 8，规则版）：每集必须申报 ≥1 new_fact 与
  ≥1 state_change（没有新事实的一集是复述集）+ 与前 N 集字符 4-gram 重复度
  > max_repeat_ratio 硬失败（拦 E19/E21/E29 三集复述；embedding 深查属 M5）。
- M4 可拍性门 `producibility`（门 12）：每集具名角色/新角色/场景上限、
  剧情关键汉字上屏硬失败（必须转图案/符号/后期贴字）、镜头语言禁入散文、
  群演全剧配额、夜景/水汽/儿童/动物成本标签（warn 进风险清单）。
- M4 品牌安全门 `compliance`（门 11）：分级词表逐句扫描——hard 红线硬失败
  （官商往来/特权炫耀/迷信），flag 绝对化用语 warn 进人审清单；词表随剧目进
  canon `config.compliance`（默认取材 E14/E26/E27/E28 已发生缺陷）。
- M4 营销门 `brand`（门 10）：排期集必须申报主卖点（缺申报 = 该集白卖）+
  牌类道具材质漂移检测（材质字必须 ∈ tiers，拦"令牌材质四套并存"）。
- M4 弧线门 `arc`（门 9）：主角权限等级 0→max_level——首次申报必须 0 起步
  （拦"开局即内部人"）、每 min_eps_per_step 集最多 +1（拦"E7 就当老板"）、
  升级必须填 cost（她付出了什么）、结局零成长 warn。
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
