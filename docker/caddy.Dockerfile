ARG CADDY_VERSION=2

FROM --platform=$BUILDPLATFORM caddy:${CADDY_VERSION}-builder AS builder

ARG TARGETOS
ARG TARGETARCH

ARG GOPROXY=https://proxy.golang.org,direct
ENV GOPROXY=$GOPROXY

RUN GOOS=$TARGETOS GOARCH=$TARGETARCH xcaddy build \
    --with github.com/zhangjiayin/caddy-geoip2 \
    --with github.com/caddy-dns/cloudflare \
    --with github.com/caddyserver/transform-encoder

FROM caddy:${CADDY_VERSION}-alpine

COPY --from=builder /usr/bin/caddy /usr/bin/caddy