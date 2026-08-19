"""Pass 1：一致性巡检（LLM 版）——只输出 diff 建议，不许重写全文。

规则版（Go 的 gates/consistency 等）拦"机器可判"的冲突；本 pass 是语义兜底：
人名漂移、道具规则矛盾、相识关系错误、引文凭空、时间线矛盾的"近似"形态。
prompt 见 baml_src/songguard.baml 的 ConsistencySweep（版本化 = git）。
"""

from __future__ import annotations

from typing import Any

PASS_ID = "sweep"


def run(client: Any, payload: dict) -> dict:
    canon_digest = payload.get("canon_digest", "")
    episodes = payload.get("episodes", [])
    from . import format_episodes

    report = client.ConsistencySweep(
        canon_digest=canon_digest,
        episodes=format_episodes(episodes),
    )
    return {
        "pass": PASS_ID,
        "findings": [f.model_dump() for f in report.findings],
    }
