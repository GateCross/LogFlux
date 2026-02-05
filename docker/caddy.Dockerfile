ARG TARGETPLATFORM=linux/amd64
FROM --platform=${TARGETPLATFORM} caddy:builder AS builder

ARG HTTP_PROXY
ARG HTTPS_PROXY
ARG NO_PROXY
ARG GOPROXY=https://proxy.golang.org,direct
ARG GOSUMDB=sum.golang.org
ENV HTTP_PROXY=$HTTP_PROXY HTTPS_PROXY=$HTTPS_PROXY NO_PROXY=$NO_PROXY \
    GOPROXY=$GOPROXY GOSUMDB=$GOSUMDB

RUN apk add --no-cache git

COPY docker/caddy.modules.txt /tmp/caddy.modules.txt

RUN set -eux; \
    modules="$(sed '/^\s*#/d;/^\s*$/d' /tmp/caddy.modules.txt | xargs -I {} printf -- "--with %s " "{}")"; \
    xcaddy build ${modules}

FROM --platform=${TARGETPLATFORM} caddy:2-alpine
COPY --from=builder /usr/bin/caddy /usr/bin/caddy
