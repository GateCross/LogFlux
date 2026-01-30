# LogFlux Docker 部署指南

## 前置要求

- Docker 20.10+
- Docker Compose 2.0+
- PostgreSQL 数据库 (外部)
- Redis (可选,外部)

## 架构说明

### 技术栈

- **基础镜像**: Alpine Linux 3.21 (稳定版)
- **前端服务器**: Caddy 2 (带 GeoIP2、Cloudflare DNS、Transform Encoder 模块)
- **后端**: Go-Zero API (Go 1.23)
- **前端**: Vue 3 (自动构建)
- **进程管理**: Supervisor

### 容器架构

```
┌─────────────────────────────────────────┐
│         LogFlux Container               │
│  ┌───────────────────────────────────┐  │
│  │       Supervisor (root)           │  │
│  │  ┌─────────────┬───────────────┐  │  │
│  │  │   Caddy     │   Backend API │  │  │
│  │  │  (logflux)  │   (logflux)   │  │  │
│  │  │   :80/:443  │     :8888     │  │  │
│  │  └─────────────┴───────────────┘  │  │
│  └───────────────────────────────────┘  │
└─────────────────────────────────────────┘
```

**特性**:
- ✅ 非 root 用户运行 (logflux:logflux, UID/GID: 1000)
- ✅ 自动重启 (Supervisor 监控)
- ✅ 前端自动构建 (无需手动 pnpm build)
- ✅ 健康检查 (30 秒间隔)
- ✅ 多阶段构建 (优化镜像大小)

## 部署步骤

### 1. 准备 GeoIP2 数据库 (可选)

如果需要使用 GeoIP2 功能进行地理位置识别：

```bash
# 下载 GeoLite2-City 数据库
cd docker
wget https://git.io/GeoLite2-City.mmdb

# 或从官方下载（需要免费注册）
# https://dev.maxmind.com/geoip/geolite2-free-geolocation-data
```

如果不需要 GeoIP2 功能：
- 注释掉 `docker-compose.yml` 中的 GeoIP2 volume 映射
- 注释掉 `Caddyfile` 中的 `import geoip` 行

### 2. 配置后端

编辑 `backend/etc/config.yaml`,配置数据库和 Redis 连接:

```yaml
Name: logflux-api
Host: 0.0.0.0
Port: 8888
Auth:
  AccessSecret: "your-secret-key"
  AccessExpire: 86400
Database:
  Host: "your-postgres-host"
  Port: 5432
  User: "postgres"
  Password: "your-password"
  DBName: "logflux"
  SSLMode: "disable"
Redis:
  Host: "your-redis-host"  # 如果不使用 Redis,留空
  Port: 6379
  Password: ""
  DB: 0
CaddyLogPath: "/var/log/caddy/access.log"
Archive:
  Enabled: true
  RetentionDay: 90
  ArchiveTable: "caddy_logs_archive"
```

### 3. 部署应用

#### 使用脚本部署 (推荐)

```bash
chmod +x docker/deploy.sh
./docker/deploy.sh
```

#### 手动部署

```bash
cd docker

# 构建镜像（包含前端自动构建）
docker compose build

# 启动容器
docker compose up -d

# 查看日志
docker compose logs -f
```

> **注意**: Docker 会自动构建前端，无需手动运行 `pnpm build`

## 服务管理

### 查看状态

```bash
docker compose ps
```

### 查看日志

```bash
# 查看所有日志
docker compose logs -f

# 查看特定服务日志
docker compose logs -f logflux

# 进入容器查看 supervisor 日志
docker compose exec logflux cat /var/log/supervisor/supervisord.log
```

### 重启服务

```bash
# 重启整个容器
docker compose restart

# 重启容器内的特定服务（需进入容器）
docker compose exec logflux supervisorctl restart backend
docker compose exec logflux supervisorctl restart caddy
```

### 停止服务

```bash
docker compose down
```

### 更新部署

```bash
# 1. 拉取最新代码
git pull

# 2. 重新构建镜像（前端会自动重新构建）
docker compose build

# 3. 重启容器
docker compose down
docker compose up -d
```

## 端口说明

- **80**: HTTP 服务
- **443**: HTTPS 服务 (需配置 SSL 证书)
- **8888**: 后端 API (容器内部,不对外暴露)

## 数据持久化

以下数据通过 Docker volumes 持久化:

- `caddy_data`: Caddy 数据目录
- `caddy_config`: Caddy 配置目录
- `caddy_logs`: Caddy 日志目录

## SSL/TLS 配置

Caddy 支持自动 HTTPS。如需配置 SSL 证书,编辑 `deploy/Caddyfile`:

```caddyfile
yourdomain.com {
  # 自动 HTTPS (Let's Encrypt)
  tls internal

  # 或使用自定义证书
  # tls /path/to/cert.pem /path/to/key.pem

  # ... 其他配置
}
```

## 健康检查

容器包含健康检查端点:

```bash
curl http://localhost/api/health
```

## 故障排查

### 1. 容器无法启动

```bash
# 查看容器日志
docker compose logs logflux

# 查看构建日志
docker compose build --no-cache

# 检查配置文件
cat ../backend/etc/config.yaml
```

### 2. 服务进程崩溃

```bash
# 进入容器查看 supervisor 状态
docker compose exec logflux supervisorctl status

# 重启特定服务
docker compose exec logflux supervisorctl restart backend
docker compose exec logflux supervisorctl restart caddy

# 查看 supervisor 日志
docker compose exec logflux cat /var/log/supervisor/supervisord.log
```

### 3. 无法连接数据库

- 检查 `config.yaml` 中的数据库配置
- 确保数据库服务可访问
- 如果数据库在本机,使用 `host.docker.internal` 作为 Host

### 4. 前端无法访问或显示异常

```bash
# 检查前端是否已构建（在容器内）
docker compose exec logflux ls -la /app/frontend

# 查看 Caddy 日志
docker compose exec logflux cat /var/log/caddy/access.log

# 测试 Caddy 配置
docker compose exec logflux caddy validate --config /etc/caddy/Caddyfile
```

### 5. GeoIP2 功能异常

```bash
# 检查 GeoIP2 数据库文件是否存在
docker compose exec logflux ls -la /usr/share/GeoIP/GeoLite2-City.mmdb

# 如果文件不存在，检查 volume 映射
docker compose config
```

### 6. 进入容器调试

```bash
# 以 root 用户进入
docker compose exec -u root logflux sh

# 以 logflux 用户进入
docker compose exec logflux sh
```

## 性能优化

### 1. Caddy 配置

- 已启用 gzip 和 zstd 压缩
- 已配置静态资源缓存
- 已启用 HTTP/2

### 2. Redis 缓存

在 `config.yaml` 中启用 Redis 以提升查询性能。

### 3. 日志归档

定期归档旧日志以保持数据库性能:

```yaml
Archive:
  Enabled: true
  RetentionDay: 90  # 保留 90 天数据
```

## 安全建议

1. 修改默认密钥:
   - `Auth.AccessSecret`: 使用强随机字符串
   - 数据库密码
   - Redis 密码

2. 使用 HTTPS:
   - 配置 SSL 证书
   - 启用 HTTP 到 HTTPS 重定向

3. 限制端口访问:
   - 仅暴露必要端口 (80, 443)
   - 使用防火墙限制访问

4. 定期更新:
   - 更新 Docker 镜像
   - 更新依赖包

## 监控

### 查看 Caddy 访问日志

```bash
docker-compose exec logflux tail -f /var/log/caddy/access.log
```

### 查看容器资源使用

```bash
docker stats logflux-app
```

## 备份

### 备份 Caddy 数据

```bash
docker run --rm -v logflux_caddy_data:/data -v $(pwd):/backup alpine tar czf /backup/caddy_data_backup.tar.gz -C /data .
```

### 恢复 Caddy 数据

```bash
docker run --rm -v logflux_caddy_data:/data -v $(pwd):/backup alpine tar xzf /backup/caddy_data_backup.tar.gz -C /data
```
