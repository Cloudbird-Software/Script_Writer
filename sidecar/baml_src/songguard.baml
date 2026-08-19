// songguard M5 LLM 旁路 —— prompt 资产（SSOT）。
//
// 这里是 issue #1 §C 的 Pass 1（一致性巡检）与 Pass 3（观众模拟）的 LLM 版：
// 规则版（Go）拦"机器可判"的跨集缺陷，这两道是"语义可判"的兜底。
// prompt 的版本化管理即本文件 + git：每次改 prompt 都是一个 commit，
// 配合 sidecar/tests 的契约测试，prompt 变更可回归。
//
// BAML 的优势在这里体现为三件事：
//   1. prompt 模板与输出 schema 同文件声明（改 prompt 必须同步看 schema）；
//   2. 结构化输出由 BAML 解析保证（比裸 JSON 更抗模型抖动）；
//   3. 生成的 pydantic 类型直接成为 sidecar 的内部数据模型。

// ---------- Pass 1：一致性巡检 ----------

class SweepFinding {
  episode int @description("违规所在集号，如 11")
  position string @description("定位，如：第2句 / 第3段")
  issue string @description("与 canon 或前文冲突的具体内容（引原文）")
  suggestion string @description("diff 建议：只给替换方案，不许重写全文")
  confidence "high" | "medium" | "low" @description("把握度：与 canon 直接冲突=high，仅疑似=low")
}

class SweepReport {
  findings SweepFinding[] @description("全部发现；干净文本返回空数组而不是编造")
}

function ConsistencySweep(canon_digest: string, episodes: string) -> SweepReport {
  client SongguardLLM
  prompt #"
    你是跨集一致性审校 agent。你的唯一职责：找出正文与 canon（事实来源）之间、
    以及正文前后集之间的矛盾，输出 diff 建议。你不许重写全文，只许逐条给出
    "定位 → 问题 → 建议改法"。

    常见缺陷类型（按优先级）：
    1. 人名/地名漂移：正文出现 canon 别名表之外的名字（如 registry 里是"宁捕快"，
       正文写"渔捕快"）。
    2. 道具规则矛盾：同一道具的规则/材质/归属与 canon 或前文不一致。
    3. 相识关系错误：两人前文已有交集，后文却写"初次相识"。
    4. 引文凭空："想起某人说过某话"但前文无人说过。
    5. 时间线矛盾：季节/日期/昼夜与前文或 canon 时间轴冲突。

    判定纪律：
    - 只有能在 canon 或前文里找到反证才算冲突；找不到反证的"感觉不对"不算。
    - 每条 finding 必须引用正文原句（issue 字段），否则人审无法定位。
    - 干净就返回空 findings，宁缺勿滥。

    ===== canon 摘要（唯一事实来源） =====
    {{ canon_digest }}

    ===== 正文（按集序拼接，每集以 [E编号] 开头） =====
    {{ episodes }}

    {{ ctx.output_format }}
  "#
}

// ---------- Pass 3：观众模拟 ----------

class HookAssessment {
  episode int
  strength "strong" | "medium" | "weak" @description("结尾钩子强度：strong=想看下一集；weak=暖收无钩子")
  reason string @description("一句话理由（引结尾原句）")
}

class ReaderReport {
  hooks HookAssessment[] @description("每集结尾钩子强度")
  drop_off_prediction string @description("预测弃剧点：第几集、为什么、观众此刻看到了什么重复")
  token_rule_restate string @description("只看正文复述：令牌/会员体系的规则是什么（一句话）")
  token_rule_consistent bool @description("复述是否能给出唯一无矛盾的答案")
}

function ReaderSimulation(canon_digest: string, episodes: string) -> ReaderReport {
  client SongguardLLM
  prompt #"
    你是竖屏短剧的观众模拟器。你不做文学评论，只模拟真实观众的追剧行为：
    每集结尾是否还想点下一集、什么时候会弃剧、看完记不记得住会员规则。

    三件事：
    1. 逐集评估结尾钩子强度（strong/medium/weak）。暖收（冲突解决、
       皆大欢喜、无新悬念）= weak，观众在这里流失。
    2. 预测弃剧点：找出"同一套路第 N 次出现"的位置——观众对重复的耐受
       通常不超过 3 次。说明弃剧时观众看到的具体重复内容。
    3. 令牌复述测试：只依据正文（忽略 canon 摘要），复述令牌/会员体系的规则。
       如果正文里的规则互相矛盾（比如材质变来变去、获取条件不一致），
       如实复述矛盾并标 token_rule_consistent=false。

    ===== canon 摘要（仅用于理解名词，复述测试以正文为准） =====
    {{ canon_digest }}

    ===== 正文（按集序拼接，每集以 [E编号] 开头） =====
    {{ episodes }}

    {{ ctx.output_format }}
  "#
}
