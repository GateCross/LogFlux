# LogFlux

日志流量分析、Caddy 图形化配置与 Coraza WAF 管理系统。

一次建站、配置热加载、日志入库查询、证书统一管理——把 Caddy 的运维复杂度收进一个面板。

## 技术栈

| 层级 | 技术 |
|---|---|
| 后端 | Go 1.27、go-zero v1.9.4、GORM、PostgreSQL、Redis(可选) |
| 前端 | Vue 3.5、Vite 8、TypeScript 6、antdv-next、Pinia、Vue Router 5、Vue Query |
| 前端基底 | vue-vben-admin v5（pnpm + Turborepo monorepo） |
| 反代 / WAF | Caddy、Coraza、OWASP CRS |
| IP 定位 | ip2region |
| 部署 | Docker Compose、Makefile |

## 核心能力

**站点中心**

- 一步建站：域名 + 上游地址，两步完成，不再走多步向导
- 站点能力三态识别（`on` / `off` / `unknown`），改上游/TLS/防护只做定点替换，其余配置字节级不变
- 复杂站点（含无法归类的 `import`）原文只读，但防护开关仍可逐项覆盖

**Caddy 配置**

- 分块工作台与原文模式双视图，预览、热加载、历史版本与回滚
- 全局选项、`:80` 入口块、日志 snippet 三个单例模板可编辑与重置
- 乐观并发控制：配置被他人改动时拒绝静默覆盖

**日志**

- 每站点开关访问日志 / 错误日志，单文件滚动上限可配
- 采集入库后按 `level` / `logger` 分流，面板内分开查询
- 保留策略以数据库为准（缺省 7 天），每日凌晨清理

**WAF**

- Coraza + OWASP CRS，运行模式（关闭/仅检测/阻断）、强度（低误报/平衡/严格）、审计三档
- CRS 规则在线更新：源同步或手动上传，校验、重构布局、激活、失败自动回滚
- 拦截与检测事件查询、统计
- Coraza 引擎仅做版本检查（编译进 Caddy，升级走镜像发布）

**证书中心**

- Caddy 自动签发 + 用户上传双源合并
- 到期告警、缺口视图、一键刷新

**其他**

- IP 区域访问控制（ip2region，国家/省份粒度）
- 通知渠道（Telegram / Slack / Webhook / 邮件 / 企业微信 / 微信公众号 / Discord / 站内信）、规则、模板、发送日志
- 定时任务管理与脚本文件
- 用户管理（账号、冻结/解冻、首次启动引导建管理员，**无角色与权限体系**）

## 架构概览

```text
Client → Caddy(:80) ┬→ 前端静态资源
                    ├→ 后端(:8888) → PostgreSQL / Redis(可选)
                    └→ Coraza WAF（CRS 规则从 /config/security/active 加载）
```

单容器部署：前端、后端、Caddy 在同一镜像内。Caddy 与后端通过 Admin API（`localhost:2019`）交互，配置写入固定 `adapt → load → persist`。

## 快速开始

### Docker 部署（推荐）

1. 准备配置

```bash
cp docker/.env.example docker/.env
cp docker/config.example.yaml backend/etc/config.yaml
```

2. 修改 `backend/etc/config.yaml`，重点检查：

| 配置项 | 说明 |
|---|---|
| `Auth.AccessSecret` | JWT 密钥，**必须改** |
| `Database.*` | PostgreSQL 连接（外部库，必需） |
| `Redis.*` | 可选，留空则不用 |
| `Waf.AllowedDomains` | WAF 规则下载域名白名单 |
| `Waf.CorazaCheckProxy` | 访问 GitHub 受限时的代理 |

3. 启动

```bash
docker compose -f docker/docker-compose.yml up -d
# 或
make up
```

4. 验证

```bash
docker compose -f docker/docker-compose.yml ps
curl -f http://localhost/api/health
```

浏览器打开 `http://localhost`，首次访问会引导创建管理员账号。

使用预构建镜像时在 `docker/.env` 设置：

```env
LOGFLUX_IMAGE=ghcr.io/<owner>/<repo>:latest
```

完整部署说明见 [`docker/README.md`](docker/README.md)。

### 本地开发

前端：

```bash
cd frontend
pnpm install
pnpm run dev:antd      # 默认 http://localhost:9527，代理到 :8888
```

后端：

```bash
cd backend
go mod download
bash scripts/download-xdb.sh
go run logflux.go -f etc/config.yaml
```

`ip2region` 的 `.xdb` 数据文件不提交仓库，首次启动或编译前需执行下载脚本（或 `make download-xdb`）。`make build-backend`、Docker 与 CI 会用 `embed_ipregion` 构建标签把 xdb 嵌入二进制；普通 `go run` / `go build` 在运行时从 `backend/internal/middleware/data/` 读取。

## 开发约定

```bash
# 改完 .api 后重新生成（必须用脚本，手敲 goctl 会产生重复脚手架）
cd backend && bash scripts/gen-api.sh

# 前端类型检查
cd frontend && pnpm -F logflux-web run typecheck
```

要点：

- **API 定义入口**：`backend/api/logflux.api`，`.api` 是接口唯一真相源
- `internal/types/types.go`、`internal/handler/routes.go` 为生成文件，**禁止手改**
- Handler 只做解析与响应，用 `result.HttpResult`；业务在 `internal/logic/`
- Caddy/WAF 域的逻辑集中在 `internal/logic/caddy/`，该域没有额外的 `internal/service` 层
- **Caddyfile 文本是唯一真相源**，站点改动只做定点替换，不得整体重渲染
- **单实例架构**：没有「节点」概念，没有 `serverId`

统一响应结构：

```json
{ "code": 0, "message": "成功", "msg": "成功", "data": {} }
```

完整分层规范、错误体系、并发与日志约定见 [`contexts/context.md`](contexts/context.md)。

## 目录结构

```text
LogFlux/
├── backend/              # go-zero 后端（api、common、internal、model、scripts）
├── frontend/
│   └── apps/src/         # 业务代码（api、router、store、views、locales）
├── docker/               # Dockerfile、Compose、运行时 Caddyfile、配置模板
├── docs/                 # 架构与运维文档
├── contexts/context.md   # 开发约定与架构速查
└── Makefile
```

前端页面入口（`frontend/apps/src/views/`）：

| 菜单 | 路由 | 目录 |
|---|---|---|
| 仪表盘 | `/dashboard` | `dashboard/` |
| 站点 | `/caddy/catalog` | `caddy/catalog/`、`caddy/sites/` |
| 安全事件 | `/logs/security` | `logs/security/` |
| 证书中心 | `/caddy/certificates` | `caddy/certificates/` |
| 访问控制 | `/caddy/access` | `caddy/access/` |
| 全局设置 | `/caddy/settings` | `caddy/settings/` |
| 日志 | `/logs/*` | `logs/access/`、`logs/system/` |
| 定时任务 | `/cron` | `cron/` |
| 通知 | `/notification/*` | `notification/` |
| 系统 | `/system/user` | `system/user/` |

配置管理页（`/caddy/config`）路由保留但已从菜单隐藏，入口在「全局设置 → 高级配置」。

## 文档

- 文档索引：[`docs/README.md`](docs/README.md)
- Caddy 配置架构：[`docs/caddy-architecture.md`](docs/caddy-architecture.md)
- WAF 与规则更新：[`docs/waf.md`](docs/waf.md)
- Docker 部署：[`docker/README.md`](docker/README.md)
- 开发约定：[`contexts/context.md`](contexts/context.md)

## 常用运维命令

```bash
make status     # 容器状态与访问地址
make logs       # 跟踪日志
make restart    # 重启
make down       # 停止
make clean      # 清理容器与构建产物
```

忘记密码时用 CLI 恢复（唯一不依赖数据库 SQL 的路径）：

```bash
logflux-api reset-password -f /app/etc/config.yaml -u <username>
```

## 许可证

MIT
