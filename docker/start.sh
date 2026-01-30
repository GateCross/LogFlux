#!/bin/sh

# ===================================================
# 注意：此脚本已不再使用
# 现在通过 supervisord.conf 管理进程启动
# 保留此文件仅供手动启动或调试使用
# ===================================================

set -e

echo "Starting LogFlux Application..."
echo "WARNING: This script is deprecated. Use supervisord instead."
echo ""

# Start backend in background
echo "Starting backend API..."
/app/logflux-api -f /app/etc/config.yaml &
BACKEND_PID=$!

# Wait for backend to be ready
echo "Waiting for backend to be ready..."
sleep 5

# Start Caddy in foreground
echo "Starting Caddy..."
exec caddy run --config /etc/caddy/Caddyfile --resume
