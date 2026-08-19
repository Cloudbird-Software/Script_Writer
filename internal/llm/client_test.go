package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Cloudbird-Software/Script_Writer/internal/canon"
)

// 与 Python 侧（sidecar/tests）断言同一份契约期望：两端对 /v1/llm-check
// 协议的理解由测试强制一致。
func contract(t *testing.T) map[string]json.RawMessage {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "sidecar", "fixtures", "llm_contract.json"))
	if err != nil {
		t.Fatalf("读契约 fixture: %v", err)
	}
	var c map[string]json.RawMessage
	if err := json.Unmarshal(data, &c); err != nil {
		t.Fatalf("解析契约 fixture: %v", err)
	}
	return c
}

// mockSidecar 起一个按 fixture 应答的假 sidecar（Go 侧通路测试；
// 真侧 Python 通路测试见 sidecar/tests）。
func mockSidecar(t *testing.T, pass string) *httptest.Server {
	t.Helper()
	want := contract(t)[pass]
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/llm-check" {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		var req Request
		_ = json.NewDecoder(r.Body).Decode(&req)
		resp := map[string]any{"pass": req.Pass, "provider": "mock"}
		var m map[string]any
		_ = json.Unmarshal(want, &m)
		for k, v := range m {
			resp[k] = v
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func eps() []Episode {
	return []Episode{
		{Ep: 1, Text: "柳青眉立在柜台后，掌柜崔白把一枚木令牌放在她手心。"},
		{Ep: 2, Text: "宁捕快……他是渔捕快，管水上勾当。"},
	}
}

func TestCheckSweepAgainstContract(t *testing.T) {
	srv := mockSidecar(t, "sweep")
	rep, err := New(srv.URL).Check(context.Background(), Request{Pass: PassSweep, CanonDigest: "d", Episodes: eps()})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if rep.Provider != "mock" || len(rep.Findings) != 1 {
		t.Fatalf("报告不符：%+v", rep)
	}
	f := rep.Findings[0]
	if f.Episode != 11 || f.Confidence != "high" || !strings.Contains(f.Issue, "渔捕快") {
		t.Fatalf("finding 应与契约一致：%+v", f)
	}
}

func TestCheckReaderAgainstContract(t *testing.T) {
	srv := mockSidecar(t, "reader")
	rep, err := New(srv.URL).Check(context.Background(), Request{Pass: PassReader, CanonDigest: "d", Episodes: eps()})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if len(rep.Hooks) != 2 || rep.TokenRuleConsistent == nil || *rep.TokenRuleConsistent {
		t.Fatalf("报告应与契约一致（2 钩子 + 令牌矛盾）：%+v", rep)
	}
	if !strings.Contains(rep.DropOffPrediction, "第8集") {
		t.Fatalf("弃剧点预测不符：%s", rep.DropOffPrediction)
	}
}

func TestReportViolations(t *testing.T) {
	srv := mockSidecar(t, "reader")
	rep, err := New(srv.URL).Check(context.Background(), Request{Pass: PassReader, Episodes: eps()})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	vs := rep.Violations()
	var weak, dropOff, token int
	for _, v := range vs {
		if v.Severity != "warn" {
			t.Fatalf("LLM 违规必须全是 warn 级：%+v", v)
		}
		switch {
		case strings.Contains(v.Message, "弱钩子"):
			weak++
		case strings.Contains(v.Message, "弃剧"):
			dropOff++
		case strings.Contains(v.Message, "会员体系"):
			token++
		}
	}
	if weak != 1 || dropOff != 1 || token != 1 {
		t.Fatalf("违规转换不符：weak=%d dropOff=%d token=%d（全部=%s）", weak, dropOff, token, vs)
	}
}

func TestCheckErrors(t *testing.T) {
	ctx := context.Background()
	// 非 2xx。
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"error":"未知 pass: nope"}`, http.StatusBadRequest)
	}))
	defer srv.Close()
	if _, err := New(srv.URL).Check(ctx, Request{Pass: "nope"}); err == nil || !strings.Contains(err.Error(), "未知 pass") {
		t.Fatalf("400 应带回 sidecar 错误信息，得到：%v", err)
	}
	// 响应 pass 与请求不符。
	srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"pass":"reader","provider":"mock"}`))
	}))
	defer srv2.Close()
	if _, err := New(srv2.URL).Check(ctx, Request{Pass: PassSweep}); err == nil || !strings.Contains(err.Error(), "pass 不符") {
		t.Fatalf("pass 不符应报错，得到：%v", err)
	}
	// 不可达 → 降级违规。
	v := Unavailable(PassSweep, errDial())
	if v.Severity != "warn" || !strings.Contains(v.Message, "降级") {
		t.Fatalf("不可用降级应为 warn 且说明降级：%+v", v)
	}
}

func TestDigest(t *testing.T) {
	c, err := canon.Load("../../internal/canon/testdata/demo")
	if err != nil {
		t.Fatalf("load demo canon: %v", err)
	}
	d := Digest(c)
	for _, want := range []string{"实体注册表", "道具规则", "台词资产", "时间轴", "柳青眉"} {
		if !strings.Contains(d, want) {
			t.Fatalf("digest 缺 %q：\n%s", want, d)
		}
	}
}

func errDial() error {
	_, err := New("http://127.0.0.1:1").Check(context.Background(), Request{Pass: PassSweep})
	return err
}
