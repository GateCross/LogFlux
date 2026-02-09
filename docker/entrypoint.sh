#!/bin/sh

# 捕获信号并转发
trap 'kill ${!}; term_handler' SIGTERM SIGINT

term_handler() {
  echo "Received signal, stopping processes..."
  if [ -n "$PID_BACKEND" ]; then
    kill -SIGTERM "$PID_BACKEND" 2>/dev/null
  fi
  if [ -n "$PID_CADDY" ]; then
    kill -SIGTERM "$PID_CADDY" 2>/dev/null
  fi
  wait "$PID_BACKEND" 2>/dev/null
  wait "$PID_CADDY" 2>/dev/null
  exit 143; # 128 + 15 -- SIGTERM
}

# 启动 LogFlux 后端 (使用 logflux 用户)
echo "Starting LogFlux Backend..."
su-exec logflux /app/logflux-api -f /app/etc/config.yaml &
PID_BACKEND=$!

# 启动 Caddy (使用 logflux 用户)
# 设置必要的环境变量
export XDG_CONFIG_HOME="/config"
export XDG_DATA_HOME="/data"
export HOME="/data"

echo "Starting Caddy..."
su-exec logflux /usr/bin/caddy run --config /etc/caddy/Caddyfile --adapter caddyfile &
PID_CADDY=$!

# 等待任意一个子进程退出
wait "$PID_BACKEND"
BACKEND_STATUS=$?

# 如果后端退出了，检查是否还在运行 Caddy，如果是则杀掉
if kill -0 "$PID_CADDY" 2>/dev/null; then
  echo "Backend exited with status $BACKEND_STATUS, stopping Caddy..."
  kill -SIGTERM "$PID_CADDY"
  wait "$PID_CADDY"
fi

exit $BACKEND_STATUS
