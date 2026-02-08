ARG CADDY_VERSION=2

FROM caddy:${CADDY_VERSION}-builder AS builder

ARG GOPROXY=https://proxy.golang.org,direct
ENV GOPROXY=$GOPROXY

RUN xcaddy build \
    --with github.com/zhangjiayin/caddy-geoip2 \
    --with github.com/caddy-dns/cloudflare \
    --with github.com/caddyserver/transform-encoder

FROM caddy:${CADDY_VERSION}-alpine
COPY --from=builder /usr/bin/caddy /usr/bin/caddy
