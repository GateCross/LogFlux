# LogFlux Docker 部署指南

## 前置要求

- Docker 20.10+
- Docker Compose 2.0+
- PostgreSQL 数据库 (外部)
- Redis (可选,外部)

## 部署步骤

### 1. 构建前端

在部署前,需要先构建前端:

```bash
cd frontend
pnpm install
pnpm run build
```

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
chmod +x deploy/deploy.sh
./deploy/deploy.sh
```

#### 手动部署

```bash
# 构建镜像
docker-compose build

# 启动容器
docker-compose up -d

# 查看日志
docker-compose logs -f
```

## 服务管理

### 查看状态

```bash
docker-compose ps
```

### 查看日志

```bash
# 查看所有日志
docker-compose logs -f

# 查看特定服务日志
docker-compose logs -f logflux
```

### 重启服务

```bash
docker-compose restart
```

### 停止服务

```bash
docker-compose down
```

### 更新部署

```bash
# 1. 重新构建前端 (如果有更新)
cd frontend && pnpm run build && cd ..

# 2. 重新构建镜像
docker-compose build

# 3. 重启容器
docker-compose down
docker-compose up -d
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
docker-compose logs logflux

# 检查配置文件
cat backend/etc/config.yaml
```

### 2. 无法连接数据库

- 检查 `config.yaml` 中的数据库配置
- 确保数据库服务可访问
- 如果数据库在本机,使用 `host.docker.internal` 作为 Host

### 3. 前端无法访问

- 检查前端是否已构建: `ls -la frontend/dist`
- 查看 Caddy 日志: `docker-compose exec logflux cat /var/log/caddy/access.log`

### 4. 进入容器调试

```bash
docker-compose exec logflux sh
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
