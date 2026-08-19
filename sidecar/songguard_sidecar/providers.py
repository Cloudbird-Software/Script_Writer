"""provider 解析层：决定 BAML 函数调用打到哪里。

三种形态（优先级从高到低）：
  1. 请求体里的 ``provider`` 字段（单次调用覆盖，测试用）；
  2. 环境变量 ``SONGUARD_SIDECAR_PROVIDER``（进程级配置）；
  3. 默认 ``mock``——没有 API key 也能起服务、跑通路（深接口的可测性保障）。

取值：
  - ``mock``  : 启动内置 mock LLM（mocksrv.py），ClientRegistry 指向它；
  - ``openai``: 用 baml_src/clients.baml 的 SongguardLLM（环境变量
                SONGGUARD_LLM_BASE_URL / SONGGUARD_LLM_API_KEY / SONGGUARD_LLM_MODEL）。
"""

from __future__ import annotations

import os
from typing import Any

from baml_py import ClientRegistry

from . import mocksrv

PROVIDER_MOCK = "mock"
PROVIDER_OPENAI = "openai"
_VALID = {PROVIDER_MOCK, PROVIDER_OPENAI}

# mock 服务器进程级单例（多个请求复用；随进程退出）。
_mock_state: dict[str, Any] = {}


def resolve(request_provider: str | None) -> tuple[Any, str]:
    """返回 (baml 同步客户端, provider 名)。

    返回的客户端已经过 with_options 定制：mock 模式指向内置 mock LLM，
    openai 模式为默认客户端（由 BAML 环境变量驱动）。
    """
    name = request_provider or os.environ.get("SONGUARD_SIDECAR_PROVIDER", PROVIDER_MOCK)
    if name not in _VALID:
        raise ValueError(f"未知 provider: {name}（可选：{sorted(_VALID)}）")

    if name == PROVIDER_MOCK:
        return _mock_client(), PROVIDER_MOCK

    from .baml_client.sync_client import b

    return b, PROVIDER_OPENAI


def _mock_client():
    if "client" not in _mock_state:
        base_url, server = mocksrv.start()
        registry = ClientRegistry()
        registry.add_llm_client(
            name="MockLLM",
            provider="openai",
            options={
                "base_url": base_url,
                "api_key": "songguard-mock",
                "model": "songguard-mock",
            },
        )
        registry.set_primary("MockLLM")
        from .baml_client.sync_client import b

        _mock_state["client"] = b.with_options(client_registry=registry)
        _mock_state["server"] = server
    return _mock_state["client"]
