# LogFlux Docker 部署指南

本文档以当前仓库实现为准，覆盖：镜像构建、容器部署、WAF/Coraza 版本配置与常见运维操作。

## 1. 目录与职责

```text
docker/
  Dockerfile                 # 本地一体化构建镜像（前端+后端+Caddy）
  runtime.Dockerfile         # CI 产物拼装镜像（配合 build-artifacts）
  caddy.Dockerfile           # 自定义 Caddy（含 Coraza/CRS 模块）
  docker-compose.yml         # 部署编排（使用预构建 image）
  Caddyfile                  # Caddy 默认配置
  config.example.yaml        # 后端配置模板
  .env.example               # compose 环境变量模板
```

## 2. 前置条件

- Docker 20.10+
- Docker Compose v2+
- 外部 PostgreSQL（必须）
- 外部 Redis（可选）

## 3. 快速部署（推荐）

### 3.1 准备配置

```bash
cp docker/.env.example docker/.env
cp docker/config.example.yaml backend/etc/config.yaml
```

请至少修改：

- `backend/etc/config.yaml`
  - `Auth.AccessSecret`
  - `Database.*`
  - `Redis.*`（可留空）

### 3.2 启动

```bash
docker compose -f docker/docker-compose.yml up -d
# 或 make up
```

### 3.3 验证

```bash
docker compose -f docker/docker-compose.yml ps
curl -f http://localhost/api/health
```

默认端口：

- `80` -> HTTP
- `443` -> HTTPS

## 4. 镜像来源与本地构建

`docker-compose.yml` 默认使用：

- `LOGFLUX_IMAGE`（默认 `logflux:local`）

### 4.1 使用远端镜像（GHCR）

`docker/.env` 示例：

```env
LOGFLUX_IMAGE=ghcr.io/<owner>/<repo>:latest
```

然后：

```bash
docker compose -f docker/docker-compose.yml pull
docker compose -f docker/docker-compose.yml up -d --no-build
```

### 4.2 本地构建应用镜像

可选先构建自定义 Caddy：

```bash
docker build -f docker/caddy.Dockerfile -t logflux-caddy:local .
```

再构建 LogFlux 应用镜像：

```bash
docker build -f docker/Dockerfile \
  --build-arg CADDY_IMAGE=logflux-caddy:local \
  --build-arg CORAZA_CURRENT_VERSION=v2.1.0 \
  -t logflux:local .
```

> 若不传 `CADDY_IMAGE`，默认 `ghcr.io/gatecross/logflux-caddy:latest`。

## 5. WAF 与 Coraza（当前行为）

### 5.1 CRS 与 Coraza 的边界

- **CRS**：支持更新源管理、同步、上传、激活、回滚。
- **Coraza 引擎**：不走“更新源配置”，仅支持版本检查（GitHub Release），不支持在线替换引擎。

### 5.2 WAF 文件目录

工作目录固定：`/config/security`

关键子目录：

- `/config/security/packages`
- `/config/security/releases`

compose 已默认持久化：

- `security_data:/config/security`

### 5.3 Coraza 版本来源与优先级

后端“当前版本”读取优先级：

1. `Waf.CorazaCurrentVersion`（配置文件）
2. `CORAZA_CURRENT_VERSION`（运行时环境变量）
3. `/app/etc/coraza-current-version`（镜像构建写入）

“最新版本”来源：

- `Waf.CorazaReleaseAPI`（默认 `https://api.github.com/repos/corazawaf/coraza-caddy/releases/latest`）

### 5.4 GitHub Release 检查代理

可通过以下方式配置代理：

- `Waf.CorazaCheckProxy`（配置文件）
- `CORAZA_CHECK_PROXY`（环境变量）

检查逻辑：**优先代理，请求失败自动回退直连**。

## 6. 配置文件示例（WAF 段）

`backend/etc/config.yaml` / `docker/config.example.yaml`：

```yaml
Waf:
  WorkDir: "/config/security"
  FetchTimeoutSec: 180
  MaxPackageBytes: 104857600
  AllowedDomains: ["github.com", "api.github.com"]
  ExtractMaxFiles: 5000
  ExtractMaxTotalBytes: 536870912
  ActivateTimeoutSec: 30
  CorazaReleaseAPI: "https://api.github.com/repos/corazawaf/coraza-caddy/releases/latest"
  CorazaCurrentVersion: ""
  CorazaCheckProxy: ""
```

## 7. CI（GitHub Actions）

- 主镜像：`.github/workflows/build-and-push.yml`
- Caddy 镜像：`.github/workflows/build-caddy.yml`

建议设置 Repo Variables：

- `REGISTRY`
- `IMAGE_NAME`
- `PLATFORM`
- `CADDY_IMAGE`
- `CORAZA_CURRENT_VERSION`（用于构建注入 `/app/etc/coraza-current-version`）

## 8. Caddy 配置生效机制

- 启动优先 `caddy run --resume`（读取 `/config/caddy/autosave.json`）
- 无 autosave 时回退 `/etc/caddy/Caddyfile`
- 后台保存配置会调用 Caddy Admin API `/adapt` + `/load`，属于热重载，无需重启容器

## 9. 常用运维命令

```bash
# 查看状态
docker compose -f docker/docker-compose.yml ps

# 查看日志
docker compose -f docker/docker-compose.yml logs -f

# 重启
docker compose -f docker/docker-compose.yml restart

# 停止并删除容器
docker compose -f docker/docker-compose.yml down
```

## 10. GeoIP2（可选）

如果不使用 GeoIP2：

- 注释 `docker/docker-compose.yml` 中 `GeoLite2-City.mmdb` 挂载
- 注释 `docker/Caddyfile` 中相关 `geoip` 引用

## 11. 常见问题

### Q1：容器起来了但 `/api/health` 不通

优先检查：

- `backend/etc/config.yaml` 的数据库连接是否可达
- `docker compose logs` 中 `logflux-api` 启动报错

### Q2：Coraza 最新版本检查超时

- 配置 `Waf.CorazaCheckProxy` 或 `CORAZA_CHECK_PROXY`
- 确认目标域名 `api.github.com` 可访问

### Q3：重启后 CRS 版本切换丢失

确认 `docker-compose.yml` 中已挂载：

- `security_data:/config/security`
