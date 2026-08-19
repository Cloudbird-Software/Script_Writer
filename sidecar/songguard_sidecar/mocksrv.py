"""内置 mock LLM 服务器（OpenAI chat-completions 兼容）。

provider=mock 时由 providers.py 启动：BAML 渲染 prompt → POST 到本服务器 →
本服务器返回确定性 JSON → BAML 解析为结构化类型。这样即便没有真实 API，
也完整走通了 BAML 的渲染/解析管线（而不是绕过它直接造对象）。

响应按 prompt 关键词路由到对应 pass 的固定响应（确定性：同输入同输出，
Go 侧与 Python 侧的契约测试都依赖这一点）。
"""

from __future__ import annotations

import json
import threading
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

# ---- 确定性响应表（与 issue #1 的真实缺陷对应，供通路测试断言） ----

SWEEP_RESPONSE = {
    "findings": [
        {
            "episode": 11,
            "position": "第1句",
            "issue": "『宁捕快……他是渔捕快』同句出现两个名字，registry 中 canonical_name 为宁捕快",
            "suggestion": "改为『宁捕快……他管水上勾当』",
            "confidence": "high",
        }
    ]
}

READER_RESPONSE = {
    "hooks": [
        {"episode": 4, "strength": "weak", "reason": "暖收：客人道谢离店，无新悬念"},
        {"episode": 7, "strength": "medium", "reason": "柳青眉立下规矩，冲突半开"},
    ],
    "drop_off_prediction": "第8集前后：『客人被热水震惊』第4次出现，观众耐受耗尽",
    "token_rule_restate": (
        "正文里令牌规则前后不一：E2 是满三名回头客赠牌，E12 变成抵押赎回，"
        "E30 变成进门领、出门交"
    ),
    "token_rule_consistent": False,
}


def _route(prompt: str) -> dict:
    """按 prompt 关键词选择响应（BAML 请求体不带函数名，以 prompt 特征路由）。"""
    if "一致性审校" in prompt:
        return SWEEP_RESPONSE
    if "观众模拟" in prompt:
        return READER_RESPONSE
    return {"error": "mock: 未知 pass"}


class _Handler(BaseHTTPRequestHandler):
    server_version = "songguard-mock-llm/0.1"

    def do_POST(self):  # noqa: N802（http.server 约定）
        length = int(self.headers.get("Content-Length", 0))
        req = json.loads(self.rfile.read(length))
        prompt = json.dumps(req.get("messages", []), ensure_ascii=False)
        content = json.dumps(_route(prompt), ensure_ascii=False)

        body = {
            "id": "chatcmpl-songguard-mock",
            "object": "chat.completion",
            "created": 0,
            "model": req.get("model", "songguard-mock"),
            "choices": [
                {
                    "index": 0,
                    "message": {"role": "assistant", "content": content},
                    "finish_reason": "stop",
                }
            ],
            "usage": {"prompt_tokens": 1, "completion_tokens": 1, "total_tokens": 2},
        }
        data = json.dumps(body).encode()
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(data)))
        self.end_headers()
        self.wfile.write(data)

    def log_message(self, *_args) -> None:  # 静音访问日志
        pass


def start() -> tuple[str, ThreadingHTTPServer]:
    """在随机本地端口启动 mock LLM，返回 (base_url, server)。

    调用方负责 server.shutdown()；server 挂在守护线程上。
    """
    srv = ThreadingHTTPServer(("127.0.0.1", 0), _Handler)
    threading.Thread(target=srv.serve_forever, daemon=True).start()
    return f"http://127.0.0.1:{srv.server_address[1]}/v1", srv
