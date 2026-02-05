# LogFlux Docker 部署（含 GitHub Actions）

## 目标与约束

- **平台**: 仅 `linux/amd64` (x86_64)
- **后端**: Go
- **前端**: Node
- **反向代理**: Caddy（自定义模块通过独立镜像构建）
- **构建方式**: GitHub Actions + GHCR（默认）

## 目录结构

```
docker/
  Dockerfile                 # 应用镜像构建（Go + Node + Caddy）
  caddy.Dockerfile           # Caddy 自定义模块构建
  caddy.modules.txt          # Caddy 模块清单
  docker-compose.yml          # 应用部署
  Caddyfile                   # Caddy 前端反向代理
  supervisord.conf            # 进程管理
  config.example.yaml         # 后端配置示例
  .env.example                # 应用部署环境变量示例
  deploy.sh                   # 应用部署脚本
```

## 前置条件

- Docker 20.10+
- Docker Compose 2.0+
- **仅 x86_64 服务器/虚拟机**（`linux/amd64`）
- PostgreSQL / Redis 为外部依赖（按 `backend/etc/config.yaml` 配置）

## 应用镜像说明

### 技术栈

- **基础镜像**: Alpine Linux 3.21 (稳定版)
- **前端服务器**: Caddy 2（带 GeoIP2、Cloudflare DNS、Transform Encoder 模块）
- **后端**: Go-Zero API（Go 1.25.3）
- **前端**: Vue 3（自动构建）
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

## 1) 构建镜像（GitHub Actions）

### 1.1 工作流文件

- 主应用镜像：`.github/workflows/build-and-push.yml`
- Caddy 镜像：`.github/workflows/build-caddy.yml`（仅在 Caddy 相关文件变更时触发）

### 1.2 可选变量（GitHub Repo Variables）

可在 GitHub 仓库 Variables 中覆盖默认值：

- `REGISTRY`（默认 `ghcr.io`）
- `IMAGE_NAME`（默认 `owner/repo`）
- `PLATFORM`（默认 `linux/amd64`）
- `CADDY_IMAGE_NAME`（默认 `gatecross/logflux-caddy`）

### 1.3 镜像标签

- `latest`（仅默认分支）
- `short-sha`

## 2) 部署应用

### 2.1 使用本地构建

```bash
cp docker/.env.example docker/.env

# 若需要本地构建 Caddy 自定义镜像（可选）
docker build -f docker/caddy.Dockerfile -t logflux-caddy:local .

docker compose -f docker/docker-compose.yml build

docker compose -f docker/docker-compose.yml up -d
```

### 2.2 使用 GHCR 镜像

在 `docker/.env` 设置：

```
LOGFLUX_IMAGE=ghcr.io/<owner>/<repo>:latest
```

如需自定义 Caddy 镜像（可选），构建时传入：

```
CADDY_IMAGE=ghcr.io/gatecross/logflux-caddy:latest
```

若镜像为私有，需要先登录：

```bash
docker login ghcr.io
```

然后：

```bash
docker compose -f docker/docker-compose.yml pull

docker compose -f docker/docker-compose.yml up -d --no-build
```

### 2.3 GeoIP2（可选）

如需地理位置识别：

```bash
cd docker
wget https://git.io/GeoLite2-City.mmdb
```

不需要 GeoIP2 时：
- 注释 `docker/docker-compose.yml` 中的 GeoIP2 volume
- 注释 `docker/Caddyfile` 中的 `import geoip` 行

## 端口说明

- **80/443**: LogFlux（Caddy）

## 备注

- Caddy 作为前端反向代理，配置见 `docker/Caddyfile`
- Dockerfile 与 Compose 已固定 `linux/amd64` 平台，不支持 ARM
