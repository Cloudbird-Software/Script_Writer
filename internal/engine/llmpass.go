package engine

import (
	"context"
	"time"

	"github.com/Cloudbird-Software/Script_Writer/internal/canon"
	"github.com/Cloudbird-Software/Script_Writer/internal/llm"
	"github.com/Cloudbird-Software/Script_Writer/internal/state"
)

// llmPassTimeout 单个 LLM pass 的调用上限（旁路慢不能拖死主流程）。
const llmPassTimeout = 120 * time.Second

// RunLLMPass 执行 M5 LLM 软 pass（Pass 1 sweep + Pass 3 reader，ADR-0003）：
// 规则门禁之后的语义兜底。结论一律 warn 级建议，永不阻断交付；
// sidecar 不可用时降级为单条可见 warn，主流程不受影响。
// client 为 nil（未配置旁路）时直接跳过；provider 非空时透传 sidecar 覆盖默认。
func RunLLMPass(client *llm.Client, provider string, c *canon.Canon, eps []state.Episode) []state.Violation {
	if client == nil {
		return nil
	}
	req := llm.Request{
		CanonDigest: llm.Digest(c),
		Episodes:    make([]llm.Episode, len(eps)),
		Provider:    provider,
	}
	for i, ep := range eps {
		req.Episodes[i] = llm.Episode{Ep: ep.Ep, Text: ep.Text}
	}

	var vs []state.Violation
	for _, pass := range []string{llm.PassSweep, llm.PassReader} {
		req.Pass = pass
		ctx, cancel := context.WithTimeout(context.Background(), llmPassTimeout)
		rep, err := client.Check(ctx, req)
		cancel()
		if err != nil {
			vs = append(vs, llm.Unavailable(pass, err))
			continue
		}
		vs = append(vs, rep.Violations()...)
	}
	return vs
}
