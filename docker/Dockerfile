# ==========================================
# Global build args (must be before first FROM)
# ==========================================
ARG CADDY_IMAGE=ghcr.io/gatecross/logflux-caddy:latest

# ==========================================
# Stage 1: Build Frontend
# ==========================================

FROM node:20-alpine AS frontend-builder

WORKDIR /app

# 让前端在 Docker 内通过 Caddy 反代访问后端
ARG VITE_SERVICE_BASE_URL=/api
ENV VITE_SERVICE_BASE_URL=$VITE_SERVICE_BASE_URL

# 跳过 git hooks，并固定 pnpm 版本（避免 corepack 拉取不确定版本）
ENV SKIP_SIMPLE_GIT_HOOKS=1
ARG PNPM_VERSION=9.15.4
RUN corepack enable && corepack prepare pnpm@${PNPM_VERSION} --activate

# 复制依赖文件（包含 workspace 定义与子包）
COPY frontend/package.json frontend/pnpm-lock.yaml frontend/pnpm-workspace.yaml ./
COPY frontend/packages/ ./packages/

# 安装依赖（利用缓存）
RUN pnpm install --frozen-lockfile

# 复制前端源码
COPY frontend/ ./

# 构建前端
RUN pnpm run build

# ==========================================
# Stage 2: Caddy binary (prebuilt)
# ==========================================
FROM ${CADDY_IMAGE} AS caddy-binary

# ==========================================
# Stage 3: Build Backend
# ==========================================
FROM golang:1.25.3-alpine AS backend-builder
ARG TARGETARCH

ARG GOPROXY=https://proxy.golang.org,direct
ENV GOPROXY=$GOPROXY

WORKDIR /app

# 复制 go mod 文件并下载依赖（利用缓存）
COPY backend/go.mod backend/go.sum ./
RUN go mod download

# 复制后端源码
COPY backend/ ./

# 构建后端
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} \
    go build -trimpath -buildvcs=false -a -installsuffix cgo -ldflags="-w -s" -o logflux-api .

# ==========================================
# Stage 4: Final Production Image
# ==========================================
FROM alpine:3.21

# 设置环境变量
ENV TZ=Asia/Shanghai \
    APP_USER=logflux \
    APP_GROUP=logflux \
    APP_UID=1000 \
    APP_GID=1000

# 安装运行时依赖并创建用户
# 合并 mkdir 和 chown 操作减少层数
# 移除 supervisor, 添加 su-exec
RUN apk --no-cache add \
    ca-certificates \
    tzdata \
    curl \
    su-exec && \
    addgroup -g ${APP_GID} ${APP_GROUP} && \
    adduser -D -u ${APP_UID} -G ${APP_GROUP} ${APP_USER} && \
    mkdir -p \
    /var/log/caddy \
    /data/caddy \
    /config/caddy \
    /app/etc && \
    chown -R ${APP_USER}:${APP_GROUP} \
    /app \
    /var/log/caddy \
    /data/caddy \
    /config/caddy

WORKDIR /app

# 使用 --chown 避免额外的 chown 层
# 复制 Caddy 二进制文件
COPY --from=caddy-binary --chown=${APP_USER}:${APP_GROUP} /usr/bin/caddy /usr/bin/caddy

# 复制前端构建产物
COPY --from=frontend-builder --chown=${APP_USER}:${APP_GROUP} /app/dist /app/frontend

# 复制后端二进制文件
COPY --from=backend-builder --chown=${APP_USER}:${APP_GROUP} /app/logflux-api /app/logflux-api

# 复制 Caddyfile（可通过构建参数切换本地代理版）
ARG CADDYFILE=docker/Caddyfile
COPY --chown=${APP_USER}:${APP_GROUP} ${CADDYFILE} /etc/caddy/Caddyfile

# 复制 entrypoint
COPY --chmod=755 docker/entrypoint.sh /app/entrypoint.sh

# 复制后端配置（构建时，可通过构建参数切换）
ARG CONFIG_FILE=docker/config.example.yaml
COPY --chown=${APP_USER}:${APP_GROUP} ${CONFIG_FILE} /app/etc/config.yaml

# 暴露端口
# 80 - HTTP
# 443 - HTTPS
# 8888 - Backend API (internal)
EXPOSE 80 443 8888

# 健康检查
HEALTHCHECK --interval=60s --timeout=10s --start-period=10s --retries=3 \
  CMD curl -f http://localhost/api/health || exit 1

ENTRYPOINT ["/app/entrypoint.sh"]
