# LogFlux Docker 部署指南

本文档以当前仓库实现为准，覆盖：镜像构建、容器部署、WAF/Coraza 版本配置与常见运维操作。

## 1. 目录与职责

```text
docker/
  Dockerfile                 # 本地一体化构建镜像（前端+后端+Caddy）
  runtime.Dockerfile         # CI 产物拼装镜像（配合 build-artifacts）
  caddy.Dockerfile           # 自定义 Caddy（含 Coraza/CRS 模块）
  docker-compose.yml         # 部署编排（使用预构建 image）
  Caddyfile                  # 容器内 Caddy 默认配置（受管模板）
  entrypoint.sh              # 容器启动脚本：WAF 布局兜底、配置恢复、拉起后端与 Caddy
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

### 3.4 首次访问与账号初始化

首次部署时数据库里没有任何账号，浏览器打开 `http://localhost` 会自动进入 **初始化页面**，在那里设置管理员用户名和密码即可（密码至少 8 位，需同时包含字母和数字）。创建成功后直接进入系统，无需重启服务。

账号初始化只在「一个账号都没有」时开放，创建完成后该入口会自动关闭。

**忘记密码时**，用 CLI 重置（会生成随机密码并打印一次，同时解除账号禁用）：

```bash
docker compose -f docker/docker-compose.yml exec logflux \
  /app/logflux-api reset-password -f /app/etc/config.yaml -u admin
```

也可以在用户管理页里给其他账号改密码，但修改自己的密码需要先输入旧密码。

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
docker build -f docker/caddy.Dockerfile \
  --build-arg CORAZA_CADDY_VERSION=v2.5.0 \
  -t logflux-caddy:local .
```

再构建 LogFlux 应用镜像：

```bash
docker build -f docker/Dockerfile \
  --build-arg CADDY_IMAGE=logflux-caddy:local \
  --build-arg CORAZA_CURRENT_VERSION=v2.6.2 \
  -t logflux:local .
```

> 若不传 `CADDY_IMAGE`，默认 `ghcr.io/gatecross/logflux-caddy:latest`。

## 5. WAF 与 Coraza（当前行为）

### 5.1 CRS 与 Coraza 的边界

- **CRS**：支持更新源管理、同步、上传、激活、回滚。激活后 Caddy 会真正加载新规则。
- **Coraza 引擎**：不走“更新源配置”，仅支持版本检查（GitHub Release），不支持在线替换引擎。

### 5.2 WAF 文件目录

工作目录固定：`/config/security`

关键子目录：

- `/config/security/packages`：下载的规则包
- `/config/security/releases/<version>`：解压后的各版本规则
- `/config/security/active`：**当前生效的规则布局**，Caddyfile 从这里加载

compose 已默认持久化：

- `security_data:/config/security`

### 5.3 规则加载方式

Caddyfile 通过绝对路径从 `/config/security/active/rules/` 加载规则：

```
Include /config/security/active/rules/@coraza.conf-recommended
Include /config/security/active/rules/@crs-setup.conf.example
Include /config/security/active/rules/@owasp_crs/*.conf
```

`@` 前缀路径由 coreruleset 内嵌文件系统提供；`load_owasp_crs` 开启时 coraza-caddy
会把内嵌文件系统与 OS 文件系统合并，绝对路径落到 OS 层，因此可以加载本项目
自己管理的规则。

布局由「激活 CRS 版本」生成：`releases/<version>` 里的官方规则包会被重构成

```
active/rules/@coraza.conf-recommended   由 coraza 仓库提供，激活时补齐
active/rules/@crs-setup.conf.example
active/rules/@owasp_crs/*.conf
active/rules/@owasp_crs/*.data
```

规则文件内部以 `@pmFromFile xxx.data` 引用同目录的数据文件，因此 `.conf` 与
`.data` 必须位于同一目录，重构时二者统一放进 `@owasp_crs`。

> 布局目录为生成物，不落在 `releases/` 内，以免污染版本内容、阻碍回滚。

### 5.4 首次启动的兜底规则

全新部署或持久化卷为空时，`/config/security/active` 尚不存在，Caddy 会因找不到
规则文件而启动失败。镜像内在 `/app/etc/crs-rules` 烘焙了一份与构建期
`CORAZA_CRS_VERSION` 一致的规则，entrypoint 会在启动时将其复制到
`/config/security/active` 作为兜底。

之后在后端同步并激活 CRS 版本时，该布局会被替换为在线更新得到的规则。

### 5.5 Coraza 版本来源与优先级

后端“当前版本”读取优先级：

1. `Waf.CorazaCurrentVersion`（配置文件）
2. `CORAZA_CURRENT_VERSION`（运行时环境变量）
3. `/app/etc/coraza-current-version`（镜像构建写入）

“最新版本”来源：

- `Waf.CorazaReleaseAPI`（默认 `https://api.github.com/repos/corazawaf/coraza-caddy/releases/latest`）

### 5.6 GitHub Release 检查代理

可通过以下方式配置代理：

- `Waf.CorazaCheckProxy`（配置文件）
- `CORAZA_CHECK_PROXY`（环境变量）

检查逻辑：**优先代理，请求失败自动回退直连**。

### 5.7 简单 WAF 设置（默认入口）

前端入口：`全局设置` 页的「WAF 设置」卡片（复用 `views/caddy/shared/components/WafPanel.vue`）

默认入口只暴露常用项：

- 模式：关闭 / 仅检测 / 阻断
- 强度：低误报 / 平衡 / 严格
- 审计日志：关闭 / 相关请求 / 全量
- 请求体检查与大小限制
- 适用站点

推荐发布流程：

1. 先保存站点配置（站点页或配置管理页均可）。
2. 在「WAF 设置」卡片选择模式、强度和适用站点。
3. 点 `预览变更` 查看生成的 directives 与候选 Caddyfile。
4. 点 `应用到 Caddy`，后端执行 `/adapt + /load` 并写入配置历史。
5. 若出现异常，可在 Caddy 配置历史中回滚。

后端 API（前缀 `/api/caddy/waf`）：

- `GET /simple-config`
- `PUT /simple-config`
- `POST /simple-config/preview`
- `POST /simple-config/apply`

失败保护（当前行为）：

- 应用前先调用 Caddy `/adapt` 校验。
- `/load` 失败时不写入数据库。
- 落库失败时会尝试回滚到 `last_good` Caddy 配置。
- 会记录 `simple_waf_last_good` / `simple_waf_apply` 配置历史动作。

### 5.8 WAF 更新管理入口

WAF 规则更新的全部入口都在「全局设置」页内的两张卡片，没有独立的「安全管理」页面：

| 卡片 | 能力 |
|---|---|
| WAF 设置（`SettingsWafCard`） | 运行模式、强度、审计、请求体限制、站点绑定 |
| WAF 来源（`WafSourcesCard`） | 源增删改、检查新版本、同步下载、手动上传、Coraza 引擎版本检查 |

> 早期版本的「高级安全管理」页面（多策略、规则例外、策略绑定、误报反馈、发布/回滚）已在 2026-06 随对应接口一起删除。若在旧文档或旧截图中看到该入口，以本节为准。

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
- 无 autosave 时回退 `/etc/caddy/Caddyfile`，默认是 LogFlux 受管简化模板
- 后台保存配置会调用 Caddy Admin API `/adapt` + `/load`，属于热加载，无需重启容器
- WAF 策略发布时会优先生成完整受管 Caddyfile；未识别为 LogFlux 默认模板的自定义配置继续只替换 Coraza directives 块

### 8.1 证书中心与 DNS-01 签发（CF_API_TOKEN）

LogFlux 证书中心（菜单 `证书中心`，路由 `/caddy/certificates`）只做**声明与展示**，不做签发编排：
签发与自动续期一律由 Caddy 完成。库存由两个来源合并：

- **Caddy 自动签发**：扫描 Caddy 证书存储目录（默认 `/data/caddy/certificates`，可用
  `backend/etc/config.yaml` 的 `Caddy.CertStorageDir` 覆盖），解析 PEM 得到有效期/签发者/SAN；
- **Caddyfile 声明**：解析站点域名与 `tls` 指令得到「应有证书」清单；声明了 HTTPS 但卷里
  没有对应文件时为 `missing`，有声明无真值的行保留为 `pending`。

上传的证书写入 `/config/certs/<主域名>/`（`fullchain.pem` + `privkey.pem`，0600）。
单实例后不再按节点分目录；存量 `/config/certs/<serverID>/<域名>/` 由库存的 `StoragePath` 定位，
仍可正常删除。该目录已通过 `cert_certs` 命名卷持久化（见 `docker-compose.yml`），重建容器不丢失。

**DNS-01（Cloudflare）签发的 token 注入**：在建站/站点配置里选择 DNS 签发后，
生成的 Caddyfile 会引用环境变量占位 `{$CF_API_TOKEN}`，token 只经环境变量进入容器，
**不写数据库、不进前端**。注入方法（任选其一）：

```yaml
# 方法一：docker-compose.yml 的 logflux 服务下追加
services:
  logflux:
    environment:
      - CF_API_TOKEN=你的 Cloudflare API Token
```

```bash
# 方法二：docker/.env 里写入后 compose 自动注入（需在 compose 中引用变量）
#   environment:
#     - CF_API_TOKEN=${CF_API_TOKEN:-}
CF_API_TOKEN=你的 Cloudflare API Token
```

说明：

- Token 需要 Cloudflare 的 `Zone - DNS - Edit` 权限，建议使用仅限目标域的受限 Token；
- token 未注入时，DNS 签发的站点会签发失败（Caddy 日志可见），HTTP-01/TLS-ALPN 自动签发不受影响；
- 需要泛域名证书（`*.example.com`）或源站无公网 80 端口时才需要 DNS-01；
- 证书到期前 14/7 天发 warning、1 天与已过期发 critical 通知（通知规则里选"证书事件"组即可订阅）。

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

默认 Caddyfile 已移除 GeoIP2 依赖，容器启动不再要求挂载 `GeoLite2-City.mmdb`。如需恢复地理位置字段，建议单独维护自定义 Caddyfile。

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

### Q4：WAF 设置应用失败（`simple-config/apply`）常见原因

优先排查：

1. Caddy 配置中缺少 `coraza_waf` 的 `directives` 块（无法注入策略）。
2. `SecAuditLogRelevantStatus` 表达式写错，导致 `/load` 加载失败。
3. `SecRequestBodyLimit` / `SecRequestBodyNoFilesLimit` 超过上限（当前限制 1 GiB）。
4. Caddy Admin API 无法访问（网络、鉴权或容器状态异常）。

处理步骤：

1. 在 UI 先点 `预览变更`，定位是 `adapt` 还是 `load` 阶段失败。
2. 看后端返回的中文错误信息与容器日志。
3. 应用中断时后端会自动回退到 `last_good`；仍异常则到 Caddy 配置历史里手动回滚。


## 12. Docker 发现标签（Caddy 配置工作台 · 可选）

LogFlux 支持按容器 labels 扫描会话候选草稿（**不**建平行 discovery DB，**不**自动 `/load`）。

在目标业务容器上打标示例：

```yaml
services:
  my-app:
    image: my-app:latest
    labels:
      logflux.enable: "true"
      logflux.host: "app.example.com"
      logflux.port: "8080"
      # 可选
      logflux.name: "My App"
      logflux.tls: "auto"           # auto | off | internal
      logflux.lb_policy: "round_robin"
      logflux.health.path: "/health"
      logflux.health.interval: "10s"
      logflux.health.timeout: "5s"
```

若 LogFlux 以容器运行，需挂载 Docker socket 才能扫描本机容器，例如：

```yaml
volumes:
  - /var/run/docker.sock:/var/run/docker.sock:ro
```

或在宿主机设置 `DOCKER_HOST=tcp://...`。扫描结果仅存管理端会话；写入草稿后仍走 Preview → 确认 → Apply_Path。
