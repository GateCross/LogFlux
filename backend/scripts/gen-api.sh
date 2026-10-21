#!/usr/bin/env bash
# 重新生成 goctl 产物（types / routes）。
#
# --style go_zero 必须显式指定：仓库文件采用 snake_case 命名
# （add_caddy_server_handler.go），而 goctl 默认的 gozero 风格会生成
# camelCase 文件（addcaddyserverhandler.go），与既有文件并存导致重复声明。
#
# 生成后只保留 internal/types 的变更；routes.go 的段落顺序在不同 goctl
# 版本间不稳定，middleware 也会被重复脚手架，两者一律还原。
set -euo pipefail

cd "$(dirname "$0")/.."

API_FILE="api/logflux.api"

goctl api go -api "$API_FILE" -dir . --style go_zero

# goctl 会补一份 middleware 脚手架，运行时的实现是 active_user_middleware.go
rm -f internal/middleware/activeuser_middleware.go
git checkout -- internal/handler/routes.go 2>/dev/null || true

# 版本号注释随本机 goctl 变化，属于噪声，固定回仓库记录的值
sed -i.bak 's|^// goctl .*$|// goctl 1.8.5|' internal/types/types.go && rm -f internal/types/types.go.bak

echo "已生成。请检查 git diff internal/types/types.go 是否只包含本次 API 变更。"
