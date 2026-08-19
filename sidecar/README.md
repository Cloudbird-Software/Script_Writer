# songguard sidecar —— M5 LLM 旁路

Prompt harness 的版本化层：**BAML 是 prompt 的唯一事实来源**（`baml_src/*.baml`，
git 即版本管理），本进程把 BAML 函数包装成一个唯一的 HTTP 端点，供 Go 主进程
（`internal/llm`）调用。规则门禁（Go）负责"机器可判"的硬阻断；本旁路只产出
**建议级**（warn）结论，永远不直接阻断交付——阻断决策留在 Go 侧。

## 深接口

对外只有两样东西：

```
python -m songguard_sidecar          # 进程入口
POST /v1/llm-check                   # 唯一端点
```

内部层次（调用方不感知）：

```
api.py            HTTP 门面（契约见模块 docstring）
 └─ passes/       LLM pass 注册表（sweep = Pass 1 一致性巡检，reader = Pass 3 观众模拟）
     └─ baml_client/   BAML 生成客户端（prompt 渲染 + 结构化解析）
         └─ providers.py / mocksrv.py   provider 解析 + 内置 mock LLM
```

## 运行

```bash
pip install -r requirements.txt
PYTHONPATH=. python -m songguard_sidecar        # 默认 127.0.0.1:8710，provider=mock
```

真实 LLM（任何 OpenAI 兼容端点）：

```bash
export SONGGUARD_SIDECAR_PROVIDER=openai
export SONGGUARD_LLM_BASE_URL=https://api.deepseek.com/v1   # 或 vLLM/Ollama/OpenAI
export SONGGUARD_LLM_API_KEY=...                            # 不落仓库
export SONGGUARD_LLM_MODEL=deepseek-chat
```

## 测试

```bash
cd sidecar && python -m unittest discover -s tests -v
```

mock provider 走的是**完整 BAML 管线**（渲染 → 内置 mock LLM → 解析），不是绕过
BAML 直接造对象；`fixtures/llm_contract.json` 是与 Go 侧 `internal/llm` 共享的
契约期望，两侧测试断言同一份文件。

## 改 prompt / 加 pass

1. 改 `baml_src/songguard.baml`（函数 + 输出 class 同文件声明，这是用 BAML 的意义）；
2. `./generate.sh` 重新生成客户端（需要 baml-cli 0.226.1，见脚本注释）——
   生成产物 `songguard_sidecar/baml_client/` **不入库**（构建产物纪律，见 .gitignore
   与 PR #9 先例），clone 后先跑一次 `./generate.sh` 再运行/测试；
3. `passes/` 下加模块并注册进 `REGISTRY`；必要时更新 `fixtures/llm_contract.json`
   与 `mocksrv.py` 的确定性响应。
