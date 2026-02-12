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

### 5.5 WAF 运行策略（P0）发布与回滚

前端入口：`安全管理` -> `运行模式`

可配置项（当前已支持）：

- `SecRuleEngine`：`On | Off | DetectionOnly`
- `SecAuditEngine`、`SecAuditLogFormat`、`SecAuditLogRelevantStatus`
- `SecRequestBodyAccess`、`SecRequestBodyLimit`、`SecRequestBodyNoFilesLimit`

推荐发布流程：

1. 先在 `运行模式` 页签点 `预览`，确认生成的 directives。
2. 点 `校验`（仅调用 `/adapt`，不生效）。
3. 点 `发布`（调用 `/adapt + /load`），`On` 模式会二次确认。
4. 若出现误拦截，可在“最近发布记录”里选择历史版本执行回滚。

后端 API（前缀 `/api/caddy/waf`）：

- `POST /policy/:id/preview`
- `POST /policy/:id/validate`
- `POST /policy/:id/publish`
- `POST /policy/rollback`
- `GET /policy/revision`

失败保护（当前行为）：

- 发布/回滚失败时会尝试自动回退到 `last_good` Caddy 配置。
- 会记录 `policy_last_good` / `policy_publish` / `policy_rollback` 配置历史动作。

### 5.6 CRS 调优模板（P1）

前端入口：`安全管理` -> `CRS 调优`

当前支持：

- 模板：`低误报 (low_fp)` / `平衡 (balanced)` / `高拦截 (high_blocking)` / `自定义`
- 字段：`tx.paranoia_level`、`tx.inbound_anomaly_score_threshold`、`tx.outbound_anomaly_score_threshold`
- 操作：`保存调优参数` -> `预览` -> `校验` -> `发布`

风险提示（当前行为）：

- 当 `PL >= 3` 时，前端会弹出高风险发布确认提示。
- 发布动作会先保存当前 CRS 调优参数，再执行策略发布，确保 revision 可追溯。

### 5.7 规则例外与策略绑定（P2）

前端入口：

- `安全管理` -> `规则例外`
- `安全管理` -> `策略绑定`

当前支持：

- 规则例外：`removeById` / `removeByTag`
- 作用域：`global` / `site(host)` / `route(path + optional method)`
- 策略绑定：按作用域和优先级管理策略生效范围

发布门禁（当前行为）：

- 若存在“同作用域 + 同优先级”的多条启用绑定冲突，发布会被阻断并返回冲突提示。

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

### Q4：策略发布失败（`publish`）常见原因

优先排查：

1. `Caddy` 配置中缺少 `coraza_waf` 的 `directives` 块（会导致无法注入策略）。
2. `SecAuditLogRelevantStatus` 表达式填写错误，导致 `/adapt` 失败。
3. `SecRequestBodyLimit` / `SecRequestBodyNoFilesLimit` 超过上限（当前限制 1 GiB）。
4. Caddy Admin API 无法访问（网络、鉴权或容器状态异常）。

处理步骤：

1. 在 UI 先执行 `校验`，定位是 `adapt` 阶段还是 `load` 阶段失败。
2. 查看 `任务日志` 和后端返回的中文错误信息。
3. 如发布中断，确认是否已自动回退到 `last_good`，必要时手动回滚发布记录。
