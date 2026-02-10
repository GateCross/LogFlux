ARG CADDY_VERSION=2
ARG CORAZA_CADDY_VERSION=v2.1.0
ARG CORAZA_CRS_VERSION=v4.23.0

FROM --platform=$BUILDPLATFORM caddy:${CADDY_VERSION}-builder AS builder

ARG TARGETOS
ARG TARGETARCH

ARG GOPROXY=https://proxy.golang.org,direct
ENV GOPROXY=$GOPROXY

RUN GOOS=$TARGETOS GOARCH=$TARGETARCH xcaddy build \
    --with github.com/corazawaf/coraza-caddy/v2@${CORAZA_CADDY_VERSION} \
    --with github.com/corazawaf/coraza-coreruleset/v4@${CORAZA_CRS_VERSION} \
    --with github.com/zhangjiayin/caddy-geoip2 \
    --with github.com/caddy-dns/cloudflare \
    --with github.com/caddyserver/transform-encoder

FROM caddy:${CADDY_VERSION}-alpine

COPY --from=builder /usr/bin/caddy /usr/bin/caddy
