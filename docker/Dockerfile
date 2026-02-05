# ==========================================
# Global build args (must be before first FROM)
# ==========================================
ARG CADDY_IMAGE=ghcr.io/gatecross/logflux-caddy:latest
ARG TARGETPLATFORM=linux/amd64
ARG TARGETARCH=amd64

# ==========================================
# Stage 1: Build Frontend
# ==========================================

FROM --platform=${TARGETPLATFORM} node:20-alpine AS frontend-builder
ARG HTTP_PROXY
ARG HTTPS_PROXY
ARG NO_PROXY
ENV HTTP_PROXY=$HTTP_PROXY HTTPS_PROXY=$HTTPS_PROXY NO_PROXY=$NO_PROXY

WORKDIR /app

# 让前端在 Docker 内通过 Caddy 反代访问后端
ARG VITE_SERVICE_BASE_URL=/api
ENV VITE_SERVICE_BASE_URL=$VITE_SERVICE_BASE_URL

# 安装前端构建依赖（部分包在 alpine 上需要编译）
RUN apk add --no-cache python3 make g++

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
FROM --platform=${TARGETPLATFORM} golang:1.25.3-alpine AS backend-builder
ARG TARGETARCH
ARG HTTP_PROXY
ARG HTTPS_PROXY
ARG NO_PROXY
ARG GOPROXY=https://proxy.golang.org,direct
ARG GOSUMDB=sum.golang.org
ENV HTTP_PROXY=$HTTP_PROXY HTTPS_PROXY=$HTTPS_PROXY NO_PROXY=$NO_PROXY \
    GOPROXY=$GOPROXY GOSUMDB=$GOSUMDB

WORKDIR /app

# 安装依赖
RUN apk add --no-cache git

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
FROM --platform=${TARGETPLATFORM} alpine:3.21

# 设置环境变量
ENV TZ=Asia/Shanghai \
    APP_USER=logflux \
    APP_GROUP=logflux \
    APP_UID=1000 \
    APP_GID=1000

# 安装运行时依赖
RUN apk --no-cache add \
    ca-certificates \
    tzdata \
    curl \
    supervisor && \
    # 创建应用用户和组
    addgroup -g ${APP_GID} ${APP_GROUP} && \
    adduser -D -u ${APP_UID} -G ${APP_GROUP} ${APP_USER}

WORKDIR /app

# 复制 Caddy 二进制文件
COPY --from=caddy-binary /usr/bin/caddy /usr/bin/caddy

# 复制前端构建产物
COPY --from=frontend-builder /app/dist /app/frontend

# 复制后端二进制文件
COPY --from=backend-builder /app/logflux-api /app/logflux-api

# 复制 Caddyfile（可通过构建参数切换本地代理版）
ARG CADDYFILE=docker/Caddyfile
COPY ${CADDYFILE} /etc/caddy/Caddyfile

# 复制 supervisor 配置
COPY docker/supervisord.conf /etc/supervisord.conf

# 创建必要的目录并设置权限
RUN mkdir -p \
    /var/log/caddy \
    /data/caddy \
    /config/caddy \
    /app/etc \
    /var/log/supervisor && \
    chown -R ${APP_USER}:${APP_GROUP} \
    /app \
    /var/log/caddy \
    /data/caddy \
    /config/caddy \
    /var/log/supervisor

# 复制后端配置（构建时，可通过构建参数切换）
ARG CONFIG_FILE=docker/config.example.yaml
COPY ${CONFIG_FILE} /app/etc/config.yaml

# 修改配置文件权限
RUN chown ${APP_USER}:${APP_GROUP} /app/etc/config.yaml

# 暴露端口
# 80 - HTTP
# 443 - HTTPS
# 8888 - Backend API (internal)
EXPOSE 80 443 8888

# 健康检查
HEALTHCHECK --interval=30s --timeout=10s --start-period=10s --retries=3 \
  CMD curl -f http://localhost/api/health || exit 1

# supervisord needs root to drop privileges to APP_USER for child processes.
CMD ["/usr/bin/supervisord", "-c", "/etc/supervisord.conf"]
