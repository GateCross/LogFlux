# LogFlux

日志流量分析管理系统

## 快速开始

### 使用 Docker 部署 (推荐)

```bash
# 1. 克隆项目
git clone https://github.com/yourusername/logflux.git
cd logflux

# 2. 配置数据库和 Redis (外部)
# 确保 PostgreSQL 和 Redis 已安装并运行

# 3. 配置应用
cp deploy/config.example.yaml backend/etc/config.yaml
# 编辑 backend/etc/config.yaml,配置数据库和 Redis 连接

# 4. 一键部署
make deploy

# 或者分步执行:
make build-frontend  # 构建前端
make build-docker    # 构建 Docker 镜像
make up              # 启动容器
```

### 管理命令

```bash
make status    # 查看状态
make logs      # 查看日志
make restart   # 重启服务
make down      # 停止服务
make clean     # 清理所有
```

详细部署文档请查看 [docker/README.md](docker/README.md)

## 功能特性

- 基于 Caddy 的日志收集和分析
- GeoIP 地理位置解析
- 实时日志监控
- 日志自动归档
- Redis 缓存加速
- 用户权限管理 (RBAC)
- 响应式仪表盘

## 技术栈

### 前端
- Vue 3
- TypeScript
- Vite
- Naive UI
- ECharts

### 后端
- Go
- Go-Zero 框架
- GORM
- PostgreSQL
- Redis

### 部署
- Docker
- Caddy (反向代理 + 日志收集)

## 架构说明

```
┌─────────────────────────────────────────┐
│           Docker Container              │
│  ┌───────────────────────────────────┐  │
│  │         Caddy Server              │  │
│  │  - 反向代理                        │  │
│  │  - 静态文件服务 (前端)             │  │
│  │  - GeoIP                          │  │
│  │  - 日志收集                        │  │
│  └────────┬──────────────────┬────────┘  │
│           │                  │           │
│  ┌────────▼────────┐  ┌──────▼────────┐  │
│  │   Frontend      │  │   Backend     │  │
│  │   (Vue 3)       │  │   (Go)        │  │
│  └─────────────────┘  └───────┬───────┘  │
└────────────────────────────────┼──────────┘
                                 │
                    ┌────────────▼────────────┐
                    │  External Services      │
                    │  - PostgreSQL           │
                    │  - Redis (Optional)     │
                    └─────────────────────────┘
```

## 配置说明

主要配置文件: `backend/etc/config.yaml`

```yaml
Database:
  Host: "host.docker.internal"  # Docker 内访问主机
  Port: 5432
  User: "postgres"
  Password: "your-password"
  DBName: "logflux"

Redis:
  Host: "host.docker.internal"
  Port: 6379

Archive:
  Enabled: true
  RetentionDay: 90  # 日志保留天数
```

## API 示例

日志源管理（目录扫描间隔默认 60 秒）：

```bash
# 新增日志源
curl -X POST http://localhost:8080/api/source \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{"name":"Caddy Logs","path":"/var/log/caddy","type":"caddy","scanInterval":60}'

# 更新日志源扫描间隔
curl -X PUT http://localhost:8080/api/source/1 \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{"scanInterval":120}'

# 查询日志源列表
curl "http://localhost:8080/api/source?page=1&pageSize=20" \
  -H "Authorization: Bearer <token>"

# 删除日志源
curl -X DELETE http://localhost:8080/api/source/1 \
  -H "Authorization: Bearer <token>"
```

## 开发

### 前端开发

```bash
cd frontend
pnpm install
pnpm run dev
```

### 后端开发

```bash
cd backend
go mod download
go run logflux.go -f etc/config.yaml
```

## 许可证

MIT License

## 贡献

欢迎提交 Issue 和 Pull Request
