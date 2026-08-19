"""songguard M5 LLM 旁路（sidecar）。

深接口纪律：本包对外只暴露两样东西——
  1. ``python -m songguard_sidecar`` 进程入口；
  2. ``POST /v1/llm-check`` 唯一 HTTP 端点（api.py）。

内部层次（api → passes → baml_client → providers/mocksrv）是包私有的，
调用方（Go 主进程 / 测试）不感知。
"""

__version__ = "0.1.0"


def main() -> None:
    """进程入口：解析环境变量并启动 HTTP 服务。"""
    from . import api

    api.serve()
