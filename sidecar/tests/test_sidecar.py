"""sidecar 通路测试（unittest，零额外测试依赖）。

覆盖三层：
  1. HTTP 门面：路由/错误码/契约字段（api.py）；
  2. BAML 全链路：mock provider 下 prompt 渲染 → 内置 mock LLM → 结构化解析
     （providers.py + mocksrv.py + baml_client，不绕过 BAML）；
  3. 跨语言契约：响应与 fixtures/llm_contract.json 一致（Go 侧 internal/llm
     的测试断言同一份文件）。

运行：cd sidecar && python -m unittest discover -s tests -v
"""

from __future__ import annotations

import json
import pathlib
import sys
import threading
import unittest
import urllib.error
import urllib.request

sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent.parent))

from songguard_sidecar import api  # noqa: E402
from songguard_sidecar.passes import format_episodes  # noqa: E402

FIXTURES = pathlib.Path(__file__).resolve().parent.parent / "fixtures" / "llm_contract.json"
CONTRACT = json.loads(FIXTURES.read_text(encoding="utf-8"))

EPISODES = [
    {"ep": 1, "text": "柳青眉立在柜台后，掌柜崔白把一枚木令牌放在她手心。"},
    {"ep": 2, "text": "宁捕快……他是渔捕快，管水上勾当。"},
]


def _post(server, body: dict) -> tuple[int, dict]:
    url = f"http://127.0.0.1:{server.server_address[1]}{api.ENDPOINT}"
    data = json.dumps(body).encode()
    req = urllib.request.Request(
        url, data=data, headers={"Content-Type": "application/json"}, method="POST"
    )
    try:
        with urllib.request.urlopen(req, timeout=30) as resp:
            return resp.status, json.loads(resp.read())
    except urllib.error.HTTPError as exc:
        return exc.code, json.loads(exc.read())


class SidecarServer(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.server = api.make_server()
        threading.Thread(target=cls.server.serve_forever, daemon=True).start()

    @classmethod
    def tearDownClass(cls):
        cls.server.shutdown()
        cls.server.server_close()

    def base_payload(self, **over) -> dict:
        payload = {"pass": "sweep", "canon_digest": "宁捕快（canonical）；令牌：木/铜/银/金", "episodes": EPISODES}
        payload.update(over)
        return payload

    def with_pass(self, pass_id: str) -> dict:
        payload = self.base_payload()
        payload["pass"] = pass_id
        return payload


class TestHTTPFace(SidecarServer):
    def test_unknown_path_404(self):
        url = f"http://127.0.0.1:{self.server.server_address[1]}/nope"
        req = urllib.request.Request(url, data=b"{}", method="POST")
        with self.assertRaises(urllib.error.HTTPError) as ctx:
            urllib.request.urlopen(req, timeout=10)
        self.assertEqual(ctx.exception.code, 404)

    def test_unknown_pass_400(self):
        status, body = _post(self.server, self.with_pass("nope"))
        self.assertEqual(status, 400)
        self.assertIn("未知 pass", body["error"])

    def test_missing_pass_400(self):
        status, body = _post(self.server, {"episodes": EPISODES})
        self.assertEqual(status, 400)
        self.assertIn("pass", body["error"])

    def test_invalid_provider_400(self):
        status, body = _post(self.server, self.base_payload(provider="bedrock"))
        self.assertEqual(status, 400)
        self.assertIn("未知 provider", body["error"])


class TestMockThroughput(SidecarServer):
    """mock provider 全链路：BAML 渲染 → 内置 mock LLM → 解析 → HTTP 响应。"""

    def test_sweep_mock(self):
        status, body = _post(self.server, self.with_pass("sweep"))
        self.assertEqual(status, 200)
        self.assertEqual(body["pass"], "sweep")
        self.assertEqual(body["provider"], "mock")
        self.assertEqual(body["findings"], CONTRACT["sweep"]["findings"])

    def test_reader_mock(self):
        status, body = _post(self.server, self.with_pass("reader"))
        self.assertEqual(status, 200)
        self.assertEqual(body["pass"], "reader")
        self.assertEqual(body["provider"], "mock")
        self.assertEqual(body["hooks"], CONTRACT["reader"]["hooks"])
        self.assertEqual(body["drop_off_prediction"], CONTRACT["reader"]["drop_off_prediction"])
        self.assertEqual(body["token_rule_restate"], CONTRACT["reader"]["token_rule_restate"])
        self.assertIs(body["token_rule_consistent"], CONTRACT["reader"]["token_rule_consistent"])

    def test_mock_deterministic(self):
        """同一输入两次调用结果一致（契约测试的前提）。"""
        _, a = _post(self.server, self.with_pass("sweep"))
        _, b = _post(self.server, self.with_pass("sweep"))
        self.assertEqual(a, b)


class TestEpisodeFormatting(unittest.TestCase):
    def test_format_episodes(self):
        out = format_episodes(EPISODES)
        self.assertIn("[E1]", out)
        self.assertIn("[E2] 宁捕快", out)


if __name__ == "__main__":
    unittest.main()
