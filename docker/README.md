# LogFlux Docker 部署

## 目录结构

```
docker/
  Dockerfile                 # 应用镜像构建（Go + Node + Caddy）
  runtime.Dockerfile         # 仅运行时镜像（配合产物构建）
  caddy.Dockerfile           # Caddy 自定义模块构建
  caddy.modules.txt          # Caddy 模块清单
  docker-compose.yml         # 应用部署
  Caddyfile                  # Caddy 反向代理配置
  supervisord.conf           # 进程管理
  config.example.yaml        # 后端配置示例
  .env.example               # 部署环境变量示例
```

## 前置条件

- Docker 20.10+
- Docker Compose 2.0+
- PostgreSQL / Redis 为外部依赖（按 `backend/etc/config.yaml` 配置）

## 构建镜像（GitHub Actions）

- 主应用镜像：`.github/workflows/build-and-push.yml`
- Caddy 镜像：`.github/workflows/build-caddy.yml`（仅 Caddy 相关变更触发）

可选变量（GitHub Repo Variables）：
`REGISTRY`、`IMAGE_NAME`、`PLATFORM`、`CADDY_IMAGE_NAME`

## 部署应用

### 本地构建

```bash
cp docker/.env.example docker/.env

# 可选：本地构建 Caddy 自定义镜像
docker build -f docker/caddy.Dockerfile -t logflux-caddy:local .

docker compose -f docker/docker-compose.yml build
docker compose -f docker/docker-compose.yml up -d
```

### 使用 GHCR 镜像

在 `docker/.env` 设置：

```
LOGFLUX_IMAGE=ghcr.io/<owner>/<repo>:latest
CADDY_IMAGE=ghcr.io/gatecross/logflux-caddy:latest
```

私有镜像需先登录：

```bash
docker login ghcr.io
```

然后：

```bash
docker compose -f docker/docker-compose.yml pull
docker compose -f docker/docker-compose.yml up -d --no-build
```

## Caddy 启动与配置生效

- 容器启动时，`docker/entrypoint.sh` 会优先尝试 `caddy run --resume`。
- 若不存在恢复文件（`/config/caddy/autosave.json`），则自动回退到 `/etc/caddy/Caddyfile`。

### 配置页面保存后如何生效

- 前端保存会调用：`POST /api/caddy/server/:serverId/config`。
- 后端会先调用 Caddy Admin API `/adapt` 校验，再调用 `/load` 下发配置。
- 这属于 **热重载**（无须手动重启容器），新配置会立即生效。

### 什么时候需要“真正重启”

- 仅在升级 Caddy 二进制、插件变更等场景才建议重启容器。
- 可执行：`docker compose -f docker/docker-compose.yml restart`（或项目根目录 `make restart`）。

### 平台说明

默认 `linux/amd64`。如需 `arm64`，在 `docker/.env` 增加：

```
PLATFORM=linux/arm64
```

## GeoIP2（可选）

如需地理位置识别：

```bash
cd docker
wget https://git.io/GeoLite2-City.mmdb
```

不需要 GeoIP2：
- 注释 `docker/docker-compose.yml` 中的 GeoIP2 volume
- 注释 `docker/Caddyfile` 中的 `import geoip` 行

## 常见问题

### Can't drop privilege as nonroot user

`supervisord` 以非 root 启动时无法切换到 `user=logflux`。
请确保容器主进程为 root，不要在 `docker-compose.yml` 里设置 `user:` 为非 root。

## 端口

- 80/443: LogFlux（Caddy）
