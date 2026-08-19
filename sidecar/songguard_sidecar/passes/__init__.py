"""LLM pass 注册表：pass id → 实现。

新增一个 LLM pass 只需要：baml_src/songguard.baml 里加函数 → generate.sh
重新生成 → 本目录加一个模块并在 REGISTRY 注册。api.py 不动（深接口）。
"""

from __future__ import annotations

from typing import Any, Callable

from . import reader, sweep

# pass id → run(client, payload: dict) -> dict
REGISTRY: dict[str, Callable[[Any, dict], dict]] = {
    sweep.PASS_ID: sweep.run,
    reader.PASS_ID: reader.run,
}


def format_episodes(episodes: list[dict]) -> str:
    """把 [{ep, text}] 序列化为 BAML 函数入参：每集以 [E编号] 开头。"""
    parts = []
    for ep in episodes:
        parts.append(f"[E{ep.get('ep', '?')}] {ep.get('text', '').strip()}")
    return "\n\n".join(parts)


def run(pass_id: str, client: Any, payload: dict) -> dict:
    if pass_id not in REGISTRY:
        raise KeyError(pass_id)
    return REGISTRY[pass_id](client, payload)
