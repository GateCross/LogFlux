# LogFlux Docker 部署（含 Gitea/Registry/Actions）

## 目标与约束

- **Gitea 版本**: 1.25.1
- **私有本地 Docker Registry**: 仅绑定本机
- **平台**: 仅 `linux/amd64` (x86_64)
- **后端**: Go
- **前端**: Node
- **反向代理**: Caddy
- **构建方式**: Gitea Actions + act_runner

## 目录结构

```
docker/
  Dockerfile                 # 应用镜像构建（Go + Node + Caddy）
  docker-compose.yml          # 应用部署
  Caddyfile                   # Caddy 前端反向代理
  supervisord.conf            # 进程管理
  config.example.yaml         # 后端配置示例
  .env.example                # 应用部署环境变量示例
  deploy.sh                   # 应用部署脚本
  infra/
    docker-compose.yml        # Gitea + Registry + act_runner
    .env.example              # 基础设施环境变量示例
    gitea/
      app.ini.example         # Gitea 配置示例
    registry/
      config.yml              # Registry 配置
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

## 1) 启动基础设施（Gitea + Registry）

### 1.1 配置环境变量

```bash
cp docker/infra/.env.example docker/infra/.env
```

按需修改：
- `GITEA_ROOT_URL` 必须是浏览器可访问地址（建议带 `/` 结尾）
- `GITEA_DOMAIN` / `GITEA_SSH_PORT`
- `REGISTRY_PORT`（默认仅本机 `127.0.0.1:5000`）

### 1.2 启动 Gitea 与 Registry

```bash
docker compose -f docker/infra/docker-compose.yml up -d
```

访问 `http://localhost:3000` 完成 Gitea 初始化。

> `docker/infra/gitea/app.ini.example` 仅作参考，实际配置优先使用环境变量方式。

### 1.3 启用 act_runner

在 Gitea 管理后台获取 Runner 注册 Token 后，填入 `docker/infra/.env`：

```bash
GITEA_RUNNER_REGISTRATION_TOKEN=your-token
```

再启动 Runner：

```bash
docker compose -f docker/infra/docker-compose.yml --profile runner up -d
```

## 2) 本地私有 Registry 说明

- 默认仅绑定 `127.0.0.1:5000`，不对外暴露
- 若使用 HTTP Registry，请在 Docker daemon 中配置 `insecure-registries`（仅限本机内网用途）

## 3) 构建镜像（Gitea Actions）

act_runner 通过宿主机 Docker 构建镜像并推送到本地 Registry。建议镜像名格式：

```
localhost:5000/logflux:latest
```

### 3.1 工作流文件

工作流文件已提供：`.gitea/workflows/build-and-push.yml`

如需调整镜像地址或名称，可在 Gitea 仓库的 Actions 变量中覆盖：

- `REGISTRY`（默认 `localhost:5000`）
- `IMAGE_NAME`（默认 `logflux`）

## 4) 部署应用

### 4.1 使用本地构建

```bash
cp docker/.env.example docker/.env

docker compose -f docker/docker-compose.yml build

docker compose -f docker/docker-compose.yml up -d
```

### 4.2 使用 Registry 镜像

在 `docker/.env` 设置：

```
LOGFLUX_IMAGE=localhost:5000/logflux:latest
```

然后：

```bash
docker compose -f docker/docker-compose.yml pull

docker compose -f docker/docker-compose.yml up -d --no-build
```

### 4.3 GeoIP2（可选）

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
- **3000**: Gitea Web
- **2222**: Gitea SSH
- **5000**: 本地 Registry（仅本机）

## 备注

- Caddy 作为前端反向代理，配置见 `docker/Caddyfile`
- Dockerfile 与 Compose 已固定 `linux/amd64` 平台，不支持 ARM
