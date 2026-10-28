# LogFlux 后端架构参考

LogFlux 使用 [Go-Zero](https://go-zero.dev/) v1.9.4 + GORM + PostgreSQL。

## 目录结构 (`backend/`)

```
backend/
├── api/                        # API 定义（.api 文件）
│   ├── logflux.api             # 主入口，import 所有模块
│   ├── base.api                # 通用结构（BaseResp、IDReq）
│   ├── auth.api                # 登录 / 刷新令牌 / 初始化管理员
│   ├── user.api                # 用户管理
│   ├── dashboard.api           # 仪表盘统计
│   ├── caddy_server.api        # Caddy 配置、历史、回滚、Docker 发现
│   ├── caddy_certificate.api   # 证书中心
│   ├── caddy_global_options.api# 全局选项 / 入口块 / 日志 snippet / 保留策略
│   ├── caddy_ip_region.api     # IP 区域访问控制
│   ├── caddy_log.api           # 日志查询与站点指标
│   ├── waf.api                 # WAF 源、上传、引擎、策略、事件
│   ├── notification.api        # 通知渠道 / 规则 / 模板 / 日志
│   ├── system_log.api          # 系统日志
│   └── cron.api                # 定时任务
├── common/                     # 公共能力
│   ├── cron/                   # Cron 表达式工具
│   ├── gorm/                   # GORM 初始化
│   ├── ingest/                 # 日志摄入（caddy/、system/、shared/）
│   ├── logging/                # 日志写入
│   ├── password/               # 密码规则（HTTP 与 CLI 共用）
│   ├── redis/                  # Redis 初始化（可选组件）
│   └── result/                 # result.HttpResult 统一响应
├── internal/
│   ├── config/                 # 配置结构体
│   ├── svc/                    # ServiceContext（依赖注入 + AutoMigrate + 播种）
│   ├── handler/                # HTTP Handler（禁止放业务逻辑）
│   ├── logic/                  # 业务逻辑（auth/caddy/cron/dashboard/log/notification/user）
│   ├── middleware/             # 登录态、限流、IP 区域、响应包装
│   ├── notification/           # 通知系统（providers/ 下 8 种渠道）
│   ├── response/               # 统一响应结构
│   ├── service/                # 业务服务层（auth/cron/dashboard/log/user）
│   ├── tasks/                  # 调度器与后台任务
│   ├── types/                  # goctl 生成，禁止手改
│   ├── utils/                  # logger、safego、timeparse
│   ├── waf/                    # 包获取/校验/解压/激活/CRS 布局
│   └── xerr/                   # 统一错误码
├── model/                      # GORM 模型（caddy/common/cron/ingest/notification/system/user/waf）
├── scripts/                    # gen-api.sh、download-xdb.sh
├── etc/                        # 本地配置（gitignore）
└── logflux.go                  # 主入口 + reset-password 子命令
```

## 核心组件

### ServiceContext (`internal/svc/service_context.go`)

全局依赖注入容器，字段分四组：

| 分组 | 字段 |
|---|---|
| 配置与基础设施 | `Config`、`DB`、`Redis` |
| 后台任务 | `LogCleanupTask`、`CronScheduler`、`WafScheduler`、`CertSyncScheduler`、`IngestMgr`、`NotificationMgr` |
| 中间件 | `ActiveUser`、`RateLimit`、`IPRegionCheck`、`IPRegionMgr` |
| Model | `SystemConfigModel`、`UserModel`、`CronTaskModel`、`CronTaskFileModel`、`CaddyLogModel`、`CertificateModel`、`SystemLogModel`、`WafEventModel` |

启动顺序：`AutoMigrate` → 破坏性列清理 → 单例配置迁移/补齐 → 播种默认账号/WAF 源/默认策略 → 注册日志目录源 → 启动调度器。

### 统一响应 (`common/result/` + `internal/response/`)

- 所有 Handler 必须用 `result.HttpResult(r, w, resp, err)`
- 响应结构 `{code, message, msg, data}`，成功码 `0`；`msg` 仅为旧前端兼容
- 禁止直接 `httpx.OkJsonCtx`，禁止新造第二套响应结构

### 日志摄入 (`common/ingest/`)

- `caddy/`：解析 Caddy 访问/错误日志，按 `level` / `logger` 分流
- `system/`：系统日志
- `shared/`：归档文件识别、路径工具等共用逻辑
- `site_log_source.go`：把 `/var/log/caddy` 注册为目录源，新站点文件自动发现

### 通知系统 (`internal/notification/`)

- `manager.go`：管理器，渠道与规则从数据库加载
- `provider.go`：`NotificationProvider` 接口（`Type` / `Validate` / `Send`）
- `providers/`：webhook、email、telegram、slack、wecom、wechat_mp、discord、in_app（共 8 种）
- `rule_engine.go`：事件匹配与静默
- `worker.go`：异步发送与重试
- `template/`：消息模板与渲染

### WAF (`internal/waf/`)

- `fetcher.go` 下载、`verifier.go` 校验、`extractor.go` 安全解压
- `activator.go` 激活与回滚（切 `current` / `last_good` + `/load`）
- `store.go` 版本仓库（`packages/`、`releases/`、软链）
- `crs_layout.go` 把 release 重构成 coraza-caddy 期望的 `active/` 布局

### 后台任务 (`internal/tasks/`)

| 任务 | 频率 |
|---|---|
| `LogCleanupTask` | 每日 02:00，按 `caddy_log_retention` 清理过期日志 |
| `CertSyncScheduler` | 每日 03:40，合并 Caddy 存储卷与上传证书 |
| `CronScheduler` | 按用户配置执行脚本任务 |
| `WafScheduler` | 按 `waf_sources.schedule` 动态装载检查/同步 |

## 开发流程

1. **定义 API**：修改 `backend/api/*.api`
2. **生成代码**：
   ```bash
   cd backend
   bash scripts/gen-api.sh
   ```
   脚本内部执行 `goctl api go -api api/logflux.api -dir . --style go_zero`，并清理 goctl 的重复脚手架与 `routes.go` 顺序噪声。
3. **实现逻辑**：`internal/logic/`
4. **注册依赖**：必要时改 `internal/svc/service_context.go`
5. **验证**：`go build ./... && go vet ./...`

> Caddy 与 WAF 域的业务逻辑集中在 `internal/logic/caddy/`，该域**没有** `internal/service` 层。这是当前有效约定，不要为它再造一层 service。

## 分层红线

| 禁止 | 原因 |
|---|---|
| 手改 `internal/types/types.go`、`internal/handler/routes.go` | goctl 生成文件 |
| Handler 内写业务或直接操作 DB | 破坏分层，且无法复用 |
| 业务层自建 DB/Redis/logger | 应统一从 `ServiceContext` 取 |
| 新造第二套响应结构 | 前端只认一种 |
| 重新引入节点/`serverId`/RBAC | 对应能力已整体删除 |
