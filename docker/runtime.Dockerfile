# syntax=docker/dockerfile:1.5

ARG CADDY_IMAGE=ghcr.io/gatecross/logflux-caddy:latest
ARG TARGETPLATFORM=linux/amd64
ARG TARGETARCH=amd64

FROM ${CADDY_IMAGE} AS caddy-binary

FROM --platform=${TARGETPLATFORM} alpine:3.21
ARG TARGETARCH

ENV TZ=Asia/Shanghai \
    APP_USER=logflux \
    APP_GROUP=logflux \
    APP_UID=1000 \
    APP_GID=1000

# 合并安装依赖、创建用户、创建目录的操作
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

# 使用 --chown 直接设置权限，避免额外的层
COPY --from=caddy-binary --chown=${APP_USER}:${APP_GROUP} /usr/bin/caddy /usr/bin/caddy
COPY --chown=${APP_USER}:${APP_GROUP} build-artifacts/frontend/dist /app/frontend
COPY --chown=${APP_USER}:${APP_GROUP} build-artifacts/backend/${TARGETARCH}/logflux-api /app/logflux-api
RUN chmod +x /app/logflux-api

ARG CADDYFILE=docker/Caddyfile
COPY --chown=${APP_USER}:${APP_GROUP} ${CADDYFILE} /etc/caddy/Caddyfile

# 复制 entrypoint
COPY --chmod=755 docker/entrypoint.sh /app/entrypoint.sh

ARG CONFIG_FILE=docker/config.example.yaml
COPY --chown=${APP_USER}:${APP_GROUP} ${CONFIG_FILE} /app/etc/config.yaml

EXPOSE 80 443 8888

HEALTHCHECK --interval=60s --timeout=10s --start-period=10s --retries=3 \
  CMD curl -f http://localhost/api/health || exit 1

ENTRYPOINT ["/app/entrypoint.sh"]
