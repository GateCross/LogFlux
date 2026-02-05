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

RUN apk --no-cache add \
    ca-certificates \
    tzdata \
    curl \
    supervisor && \
    addgroup -g ${APP_GID} ${APP_GROUP} && \
    adduser -D -u ${APP_UID} -G ${APP_GROUP} ${APP_USER}

WORKDIR /app

COPY --from=caddy-binary /usr/bin/caddy /usr/bin/caddy
COPY build-artifacts/frontend/dist /app/frontend
COPY build-artifacts/backend/${TARGETARCH}/logflux-api /app/logflux-api

ARG CADDYFILE=docker/Caddyfile
COPY ${CADDYFILE} /etc/caddy/Caddyfile

COPY docker/supervisord.conf /etc/supervisord.conf

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
    /var/log/supervisor && \
    chmod +x /app/logflux-api

ARG CONFIG_FILE=docker/config.example.yaml
COPY ${CONFIG_FILE} /app/etc/config.yaml

RUN chown ${APP_USER}:${APP_GROUP} /app/etc/config.yaml

EXPOSE 80 443 8888

HEALTHCHECK --interval=30s --timeout=10s --start-period=10s --retries=3 \
  CMD curl -f http://localhost/api/health || exit 1

# supervisord needs root to drop privileges to APP_USER for child processes.
CMD ["/usr/bin/supervisord", "-c", "/etc/supervisord.conf"]
