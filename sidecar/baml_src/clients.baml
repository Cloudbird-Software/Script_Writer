// M5 LLM 旁路的客户端配置：OpenAI 兼容端点（vLLM/Ollama/DeepSeek/真实 OpenAI 均可）。
// 凭据走环境变量，不落仓库（AGENTS.md 硬规则 4）。sidecar 的 provider 层
// （songguard_sidecar/providers.py）在 mock 模式下用 ClientRegistry 把函数调用
// 指向内置 mock LLM 服务器，不触网。
client<llm> SongguardLLM {
  provider openai-generic
  options {
    base_url env.SONGGUARD_LLM_BASE_URL
    api_key env.SONGGUARD_LLM_API_KEY
    model env.SONGGUARD_LLM_MODEL
  }
}
