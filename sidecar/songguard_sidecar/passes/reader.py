"""Pass 3：观众模拟（LLM 版）——钩子强度 / 弃剧点 / 令牌复述测试。

对应 issue #1 §C Pass 3：只读视角的追更动力评估。规则版无法模拟"观众体感"，
这是天然的 LLM pass。prompt 见 baml_src/songguard.baml 的 ReaderSimulation。
"""

from __future__ import annotations

from typing import Any

PASS_ID = "reader"


def run(client: Any, payload: dict) -> dict:
    canon_digest = payload.get("canon_digest", "")
    episodes = payload.get("episodes", [])
    from . import format_episodes

    report = client.ReaderSimulation(
        canon_digest=canon_digest,
        episodes=format_episodes(episodes),
    )
    return {
        "pass": PASS_ID,
        "hooks": [h.model_dump() for h in report.hooks],
        "drop_off_prediction": report.drop_off_prediction,
        "token_rule_restate": report.token_rule_restate,
        "token_rule_consistent": report.token_rule_consistent,
    }
