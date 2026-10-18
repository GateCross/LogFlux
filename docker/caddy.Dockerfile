ARG CADDY_VERSION=2
ARG CORAZA_CADDY_VERSION=v2.6.2
ARG CORAZA_CRS_VERSION=v4.25.0

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

# 下载与编译期 coreruleset 模块同版本的规则，作为运行时的初始布局。
#
# Caddyfile 改从 /config/security/active/rules/ 加载规则，该目录由后端激活
# CRS 版本时生成；在它就绪前，entrypoint 用这里烘焙的规则兜底，避免 Caddy
# 因找不到规则文件而启动失败。
#
# 布局与 coraza-coreruleset 内嵌的 go:embed rules 保持一致：
#
#	rules/@coraza.conf-recommended   （来自 coraza 仓库，coreruleset 包内没有）
#	rules/@crs-setup.conf.example    （来自 coreruleset 发布包顶层）
#	rules/@owasp_crs/*.conf          （来自 coreruleset 发布包 rules/）
#	rules/@owasp_crs/*.data
FROM --platform=$BUILDPLATFORM alpine:3.21 AS rules-downloader

ARG CORAZA_CRS_VERSION
ARG CORAZA_CADDY_VERSION

# coraza-coreruleset 的版本号与上游 coreruleset 一致。
ARG CRS_RELEASE_VERSION=${CORAZA_CRS_VERSION}
# coreruleset.conf-recommended 由 coraza 提供；coraza-caddy v2.6.x 依赖 coraza/v3 v3.8.x。
ARG CORAZA_CONFIG_VERSION=v3.8.1

RUN set -eux; \
    apk add --no-cache curl tar; \
    mkdir -p /out/rules/@owasp_crs; \
    \
    curl -fsSL --retry 3 \
      "https://github.com/coreruleset/coreruleset/archive/refs/tags/${CRS_RELEASE_VERSION}.tar.gz" \
      -o /tmp/crs.tar.gz; \
    tar -xzf /tmp/crs.tar.gz -C /tmp; \
    SRC="$(find /tmp -maxdepth 1 -type d -name 'coreruleset-*' | head -n1)"; \
    test -n "$SRC"; \
    cp -rL "${SRC}/rules/." /out/rules/@owasp_crs/; \
    cp -L "${SRC}/crs-setup.conf.example" /out/rules/@crs-setup.conf.example; \
    rm -rf /tmp/crs.tar.gz "$SRC"; \
    \
    curl -fsSL --retry 3 \
      "https://raw.githubusercontent.com/corazawaf/coraza/${CORAZA_CONFIG_VERSION}/coraza.conf-recommended" \
      -o /out/rules/@coraza.conf-recommended; \
    \
    test -s /out/rules/@coraza.conf-recommended; \
    test -s /out/rules/@crs-setup.conf.example; \
    test -n "$(ls -A /out/rules/@owasp_crs)"; \
    ls /out/rules | head -n 5

FROM caddy:${CADDY_VERSION}-alpine

COPY --from=builder /usr/bin/caddy /usr/bin/caddy
COPY --from=rules-downloader /out/rules /app/etc/crs-rules
