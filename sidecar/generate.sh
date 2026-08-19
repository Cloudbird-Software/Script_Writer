#!/usr/bin/env bash
# 重新生成 BAML python 客户端（改 baml_src/*.baml 后运行一次，产物随仓库提交）。
# 需要 baml-cli（0.226.1）与 baml-py 同版本：
#   下载：https://github.com/BoundaryML/baml/releases（baml-cli-<ver>-x86_64-unknown-linux-gnu.tar.gz）
set -euo pipefail
cd "$(dirname "$0")"
exec baml-cli generate
