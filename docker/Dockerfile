# ==========================================
# Stage 1: Build Caddy with custom modules
# ==========================================
FROM caddy:builder AS caddy-builder

RUN xcaddy build \
    --with github.com/zhangjiayin/caddy-geoip2 \
    --with github.com/caddy-dns/cloudflare \
    --with github.com/caddyserver/transform-encoder

# ==========================================
# Stage 2: Build Backend
# ==========================================
FROM golang:1.23-alpine AS backend-builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY backend/go.mod backend/go.sum ./
RUN go mod download

# Copy backend source code
COPY backend/ ./

# Build backend
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o logflux-api .

# ==========================================
# Stage 3: Final Production Image
# ==========================================
FROM alpine:latest

# Install ca-certificates for HTTPS support
RUN apk --no-cache add ca-certificates tzdata curl

# Set timezone
ENV TZ=Asia/Shanghai

WORKDIR /app

# Copy Caddy binary from builder
COPY --from=caddy-builder /usr/bin/caddy /usr/bin/caddy

# Copy pre-built frontend (需要在构建前运行 pnpm run build)
COPY frontend/dist /app/frontend

# Copy backend binary from builder
COPY --from=backend-builder /app/logflux-api /app/logflux-api

# Copy backend config
COPY backend/etc/config.yaml /app/etc/config.yaml

# Copy Caddyfile
COPY Caddyfile /etc/caddy/Caddyfile

# Create necessary directories
RUN mkdir -p /var/log/caddy /data/caddy /config/caddy

# Expose ports
# 80 - HTTP
# 443 - HTTPS
# 8888 - Backend API (internal)
EXPOSE 80 443 8888

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD curl -f http://localhost/api/health || exit 1

# Start script
COPY start.sh /app/start.sh
RUN chmod +x /app/start.sh

CMD ["/app/start.sh"]
