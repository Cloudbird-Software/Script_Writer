"""唯一对外 HTTP 门面：POST /v1/llm-check。

契约（Go 客户端 internal/llm 与本文件双向遵守，契约测试见 sidecar/tests）：

请求：
  {
    "pass":         "sweep" | "reader",     // 必填，LLM pass id
    "provider":     "mock" | "openai",      // 可选，覆盖进程级配置
    "canon_digest": "…",                    // canon 摘要（角色表/道具规则等）
    "episodes":     [{"ep": 1, "text": "…"}] // 按集序
  }

响应（sweep）：
  {"pass": "sweep", "provider": "mock",
   "findings": [{"episode", "position", "issue", "suggestion", "confidence"}]}

响应（reader）：
  {"pass": "reader", "provider": "mock",
   "hooks": [{"episode", "strength", "reason"}],
   "drop_off_prediction": "…", "token_rule_restate": "…",
   "token_rule_consistent": false}

错误：400/500 + {"error": "…"}。LLM 输出是建议级（warn），不会造成交付阻断——
阻断决策永远在 Go 侧的规则门禁。
"""

from __future__ import annotations

import json
import os
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

from . import passes, providers

ENDPOINT = "/v1/llm-check"


class _Handler(BaseHTTPRequestHandler):
    server_version = "songguard-sidecar/0.1"

    def do_POST(self):  # noqa: N802（http.server 约定）
        if self.path != ENDPOINT:
            self._json(404, {"error": f"未知路径: {self.path}（唯一端点 {ENDPOINT}）"})
            return
        try:
            length = int(self.headers.get("Content-Length", 0))
            payload = json.loads(self.rfile.read(length) or b"{}")
        except (ValueError, json.JSONDecodeError) as exc:
            self._json(400, {"error": f"请求体不是合法 JSON: {exc}"})
            return

        pass_id = payload.get("pass")
        if not pass_id:
            self._json(400, {"error": "缺少必填字段 pass（sweep | reader）"})
            return
        if pass_id not in passes.REGISTRY:
            self._json(
                400,
                {"error": f"未知 pass: {pass_id}（可选：{sorted(passes.REGISTRY)}）"},
            )
            return

        try:
            client, provider = providers.resolve(payload.get("provider"))
            result = passes.run(pass_id, client, payload)
        except ValueError as exc:
            self._json(400, {"error": str(exc)})
            return
        except Exception as exc:  # noqa: BLE001 —— 门面层兜底，错误必须可见但不崩
            self._json(500, {"error": f"{type(exc).__name__}: {exc}"})
            return

        result["provider"] = provider
        self._json(200, result)

    def _json(self, status: int, body: dict) -> None:
        data = json.dumps(body, ensure_ascii=False).encode()
        self.send_response(status)
        self.send_header("Content-Type", "application/json; charset=utf-8")
        self.send_header("Content-Length", str(len(data)))
        self.end_headers()
        self.wfile.write(data)

    def log_message(self, *_args) -> None:  # 静音访问日志
        pass


def make_server(host: str = "127.0.0.1", port: int = 0) -> ThreadingHTTPServer:
    """构建服务实例（不启动事件循环）；测试与 serve() 共用。"""
    return ThreadingHTTPServer((host, port), _Handler)


def serve() -> None:
    """进程入口：读环境变量并阻塞服务。"""
    host = os.environ.get("SONGUARD_SIDECAR_HOST", "127.0.0.1")
    port = int(os.environ.get("SONGUARD_SIDECAR_PORT", "8710"))
    server = make_server(host, port)
    print(f"songguard sidecar 监听 http://{host}:{port}{ENDPOINT}", flush=True)
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        server.shutdown()
