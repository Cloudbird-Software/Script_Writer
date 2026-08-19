package songguard

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// M5 软 pass 门面契约：WithSidecar 启用后 LLM 建议必须以 warn 级
// 进入报告（进风险清单与人审），但永不阻断交付；sidecar 不可用时
// 降级为可见 warn，主流程照常完成。

// mockSidecarServer 起一个按共享契约 fixture 应答的假 sidecar，
// 并记录收到的请求供断言（canon 摘要与集文本必须真的送过去）。
func mockSidecarServer(t *testing.T) (*httptest.Server, *[]llmRequest) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "sidecar", "fixtures", "llm_contract.json"))
	if err != nil {
		t.Fatalf("读契约 fixture: %v", err)
	}
	var contract map[string]json.RawMessage
	if err := json.Unmarshal(data, &contract); err != nil {
		t.Fatalf("解析契约 fixture: %v", err)
	}

	var got []llmRequest
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/llm-check", func(w http.ResponseWriter, r *http.Request) {
		var req llmRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		got = append(got, req)
		resp := map[string]any{"pass": req.Pass, "provider": "mock"}
		var m map[string]any
		_ = json.Unmarshal(contract[req.Pass], &m)
		for k, v := range m {
			resp[k] = v
		}
		_ = json.NewEncoder(w).Encode(resp)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, &got
}

type llmRequest struct {
	Pass        string `json:"pass"`
	CanonDigest string `json:"canon_digest"`
	Episodes    []struct {
		Ep   int    `json:"ep"`
		Text string `json:"text"`
	} `json:"episodes"`
}

func TestFacadeLLMPassEndToEnd(t *testing.T) {
	srv, got := mockSidecarServer(t)
	g := New(WithSidecar(srv.URL), WithProvider("mock"))
	rep, err := g.Check(filepath.Join("..", "..", "examples", "demo", "manifest.yaml"))
	if err != nil {
		t.Fatalf("check demo: %v", err)
	}

	// 两个 pass 都被调用，且带上了 canon 摘要与全部集正文。
	if len(*got) != 2 {
		t.Fatalf("应调用 sweep+reader 两个 pass，实际 %d 次", len(*got))
	}
	var sweep, reader llmRequest
	for _, req := range *got {
		switch req.Pass {
		case "sweep":
			sweep = req
		case "reader":
			reader = req
		}
	}
	if sweep.Pass == "" || reader.Pass == "" {
		t.Fatalf("pass 清单不符：%+v", *got)
	}
	if !strings.Contains(sweep.CanonDigest, "实体注册表") || len(sweep.Episodes) != 5 {
		t.Fatalf("sweep 请求应带 canon 摘要与 5 集正文：%+v", sweep)
	}
	for _, req := range *got {
		if req.Episodes[0].Ep != 1 || req.Episodes[0].Text == "" {
			t.Fatalf("集正文未送达 sidecar：%+v", req.Episodes[0])
		}
	}

	// LLM 结论全部 warn 级且进入报告，但不产生硬失败、不阻断交付。
	if rep.HasError() || rep.Blocked() {
		t.Fatalf("LLM 软 pass 不得阻断交付：%s", rep.ViolationsJSON())
	}
	errs, warns, _ := rep.Counts()
	if errs != 0 {
		t.Fatalf("demo + LLM 建议不应有 error 级：%s", rep.ViolationsJSON())
	}
	if warns == 0 {
		t.Fatal("契约 fixture 含弱钩子/弃剧点/令牌矛盾，warns 不应为 0")
	}
	all := rep.AllViolations()
	for _, v := range all {
		if strings.HasPrefix(v.Gate, "llm-") && v.Severity != "warn" {
			t.Fatalf("LLM 违规必须 warn 级：%+v", v)
		}
	}
	j := rep.SummaryJSON()
	for _, want := range []string{"llm-sweep", "llm-reader", "弱钩子", "弃剧", "会员体系"} {
		if !strings.Contains(j, want) {
			t.Fatalf("摘要 JSON 缺 LLM 结论 %q：%s", want, j)
		}
	}
}

func TestFacadeLLMPassDegradedWhenSidecarDown(t *testing.T) {
	// 不可达端口：两个 pass 各降级为一条 warn，主流程照常完成。
	g := New(WithSidecar("http://127.0.0.1:1"))
	rep, err := g.Check(filepath.Join("..", "..", "examples", "demo", "manifest.yaml"))
	if err != nil {
		t.Fatalf("sidecar 挂掉不应让 check 失败: %v", err)
	}
	if rep.HasError() {
		t.Fatal("降级是 warn 级，不应产生硬失败")
	}
	var degraded int
	for _, v := range rep.AllViolations() {
		if strings.Contains(v.Message, "不可用（降级") {
			degraded++
		}
	}
	if degraded != 2 {
		t.Fatalf("sweep+reader 应各降级一条 warn，实际 %d 条：%s", degraded, rep.ViolationsJSON())
	}
}

func TestFacadeDefaultSkipsLLMPass(t *testing.T) {
	// 不配置 WithSidecar：完全不发请求（向后兼容，CI 无 Python 依赖）。
	g := New()
	rep, err := g.Check(filepath.Join("..", "..", "examples", "demo", "manifest.yaml"))
	if err != nil {
		t.Fatalf("check demo: %v", err)
	}
	for _, v := range rep.AllViolations() {
		if strings.HasPrefix(v.Gate, "llm-") {
			t.Fatalf("未启用旁路不应有 LLM 违规：%+v", v)
		}
	}
}
