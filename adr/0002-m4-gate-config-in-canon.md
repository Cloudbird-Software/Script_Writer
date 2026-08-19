# ADR-0002：M4 门禁配置进 canon 第七张表（config.yaml）+ delta 扩展

- 状态：accepted · 日期：2026-08-20 · 决策来源：issue #1 §B-2（十二道门禁 + 两道软门）

## 背景

M2 五道硬门的阈值（字数区间、悬置集数、错别字表、合规词表、镜头语言黑名单）目前
要么硬编码在门禁里，要么根本没有。M4 要落的九道门（hygiene / lineownership /
emotion / arc / brand / compliance / producibility / novelty / voice）全部带
"随剧目变化"的参数：换一部剧，错别字表、情绪类型、群演配额、合规红线全部不同。
参数写死 = 每换项目就改 Go 代码 = 资产与代码耦合。

## 决定

1. **门禁配置是资产，进 canon**：新增第七张表 `config.yaml`（`canon.Config`），
   每道门一段（format / hook_payoff / hygiene / emotion / arc / producibility /
   compliance / novelty / voice），字段名与门禁 id 一致。与六张表同目录、同版本化、
   同结构校验（`validateConfig`）。
2. **可选 + 默认值**：config.yaml 缺省合法（旧 canon 目录零改动可用）。各段零值
   经 `Config.WithDefaults()` 填充内置默认。默认词表**直接取材 issue #1 缺陷清单**
   （愣→怔、暖幢栋、镜头一抬、铺条官路、头一份……）——已发生过的缺陷默认就是黑名单。
3. **delta 协议扩展三个字段**（`state.Delta`）：
   - `arc: {level, cost}`：主角权限等级申报。台账层管单调不倒退 + 单步 ≤+1；
     升速（每 min_eps_per_step 集最多 +1）与 cost 必填归弧线门（config 阈值）。
   - `scenes: [string]`：本集场景列表，可拍性门校验 ≤ max_scenes_per_ep。
   - `crowd: bool`：群演场面申报，台账累计 CrowdEps，可拍性门按全剧配额判定。
4. **分层纪律不变**：state 只做台账层（守恒/单调/记账），文本与阈值判定在
   rules/gates 子包；阈值全部来自 `canon.Config.WithDefaults()`，门禁代码零常量。

## 备选方案与否决理由

- **硬编码进各门禁**：换项目要改代码，资产不可版本化——否决。
- **CLI flag 传参**：参数属于 canon 资产（与六张表强关联，如 arc.character 必须
  引用 entities id，需要跨表校验），flag 无法做结构校验——否决。
- **独立 config 包（Go 代码）**：客户/运营改不动 Go；YAML 是六张表已确立的资产
  形态，保持一致——否决。

## 影响

- `canon.Load` 目录里多一个可选文件；`Canon.Config` 新字段。
- 六张表 schema 未动（向后兼容）；CHANGELOG 记 Unreleased/Added。
- 后续 PR4~PR12 各门禁统一从 `c.Config.WithDefaults()` 读参，注册表一行接入。
