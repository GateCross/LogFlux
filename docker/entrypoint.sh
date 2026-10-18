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

# 准备 WAF 规则布局
#
# Caddyfile 中的 coraza_waf 从 /config/security/active/rules/ 加载规则，
# 该目录由后端激活 CRS 版本时生成。全新部署或持久化卷为空时目录不存在，
# Caddy 会因找不到规则文件而启动失败，这里先用镜像内烘焙的规则兜底。
# 之后在后端同步并激活 CRS 版本时，会被替换为在线更新得到的规则。
prepare_waf_layout() {
  RULES_SRC="/app/etc/crs-rules"
  LAYOUT_DIR="/config/security/active"
  RULES_DST="${LAYOUT_DIR}/rules"

  if [ -d "$RULES_DST/@owasp_crs" ] && [ -n "$(ls -A "$RULES_DST/@owasp_crs" 2>/dev/null)" ]; then
    echo "WAF rules layout already present, skipping bootstrap."
    return 0
  fi

  if [ ! -d "$RULES_SRC/@owasp_crs" ]; then
    echo "WARNING: bundled CRS rules not found at $RULES_SRC, WAF may fail to start."
    return 0
  fi

  echo "Bootstrapping WAF rules layout from bundled rules..."
  mkdir -p "$RULES_DST" || return 0
  cp -RL "$RULES_SRC/." "$RULES_DST/" 2>/dev/null || cp -rL "$RULES_SRC/." "$RULES_DST/" 2>/dev/null || {
    echo "WARNING: failed to copy bundled CRS rules."
    return 0
  }
  chown -R logflux:logflux /config/security 2>/dev/null || true
}

prepare_waf_layout

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
AUTOSAVE_PATH="${XDG_CONFIG_HOME}/caddy/autosave.json"

if [ -f "$AUTOSAVE_PATH" ]; then
  echo "Found Caddy autosave config at $AUTOSAVE_PATH, starting with --resume..."
  su-exec logflux /usr/bin/caddy run --resume &
else
  echo "No Caddy autosave config found, falling back to /etc/caddy/Caddyfile..."
  su-exec logflux /usr/bin/caddy run --config /etc/caddy/Caddyfile --adapter caddyfile &
fi
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
