// Package llm 是 M5 LLM 旁路的 Go 端（sidecar HTTP 门面的唯一对端）。
//
// 深接口纪律：本包对 engine/songguard 只暴露两个面——
//
//	client := llm.New("http://127.0.0.1:8710")   // sidecar 地址
//	rep, err := client.Check(ctx, llm.Request{…}) // 唯一调用
//
// 协议契约与 sidecar/songguard_sidecar/api.py 的 docstring 双向对齐，
// 两侧测试共同断言 sidecar/fixtures/llm_contract.json。LLM 结论一律
// 建议级（warn）：本包不产生 error 级违规，阻断决策永远在规则门禁。
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Pass id（与 sidecar passes.REGISTRY 一致）。
const (
	PassSweep  = "sweep"  // Pass 1：一致性巡检（diff 建议）
	PassReader = "reader" // Pass 3：观众模拟（钩子/弃剧点/令牌复述）
)

// Client 调用 songguard sidecar；零值不可用，须 New 构造。
type Client struct {
	baseURL string
	hc      *http.Client
}

// New 创建 sidecar 客户端；baseURL 形如 http://127.0.0.1:8710（不带路径）。
func New(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		hc:      &http.Client{Timeout: 120 * time.Second},
	}
}

// Episode 是喂给 LLM pass 的一集（集号 + 正文）。
type Episode struct {
	Ep   int    `json:"ep"`
	Text string `json:"text"`
}

// Request 是 POST /v1/llm-check 的请求体。
type Request struct {
	Pass        string    `json:"pass"`               // sweep | reader
	CanonDigest string    `json:"canon_digest"`       // canon 摘要（Digest 产出）
	Episodes    []Episode `json:"episodes"`           // 按集序
	Provider    string    `json:"provider,omitempty"` // 可选覆盖（mock | openai）
}

// Report 是 /v1/llm-check 的响应体（两种 pass 的字段并集，未用的为零值）。
type Report struct {
	Pass                string    `json:"pass"`
	Provider            string    `json:"provider"`
	Findings            []Finding `json:"findings,omitempty"`            // sweep
	Hooks               []Hook    `json:"hooks,omitempty"`               // reader
	DropOffPrediction   string    `json:"drop_off_prediction,omitempty"` // reader
	TokenRuleRestate    string    `json:"token_rule_restate,omitempty"`  // reader
	TokenRuleConsistent *bool     `json:"token_rule_consistent"`         // reader；指针区分"未回答"
}

// Finding 是 sweep 的一条 diff 建议（对应 sidecar 的 SweepFinding）。
type Finding struct {
	Episode    int    `json:"episode"`
	Position   string `json:"position"`
	Issue      string `json:"issue"`
	Suggestion string `json:"suggestion"`
	Confidence string `json:"confidence"` // high | medium | low
}

// Hook 是 reader 对一集结尾钩子强度的评估。
type Hook struct {
	Episode  int    `json:"episode"`
	Strength string `json:"strength"` // strong | medium | weak
	Reason   string `json:"reason"`
}

// Check 调用 sidecar 执行一个 LLM pass。非 2xx 响应返回错误（调用方决定降级）。
func (c *Client) Check(ctx context.Context, req Request) (*Report, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("llm: 序列化请求: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/llm-check", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("llm: 构造请求: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.hc.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("llm: 调用 sidecar: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var e struct {
			Error string `json:"error"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&e)
		if e.Error == "" {
			e.Error = resp.Status
		}
		return nil, fmt.Errorf("llm: sidecar %d: %s", resp.StatusCode, e.Error)
	}

	var rep Report
	if err := json.NewDecoder(resp.Body).Decode(&rep); err != nil {
		return nil, fmt.Errorf("llm: 解析响应: %w", err)
	}
	if rep.Pass != req.Pass {
		return nil, fmt.Errorf("llm: 响应 pass 不符（请求 %s，得到 %s）", req.Pass, rep.Pass)
	}
	return &rep, nil
}
