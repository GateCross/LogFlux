# LogFlux 项目上下文

> 更新：2026-10-28
> 本文件是 AI/新成员的开发约定与架构速查，描述**当前代码**。文档索引见 [`../docs/README.md`](../docs/README.md)。

## 1. 项目定位

日志流量分析、Caddy 图形化配置与 Coraza WAF 管理系统。前后端分离开发，一体化容器部署（前端静态资源 + 后端二进制 + Caddy 同一镜像）。

核心能力：

- **站点中心**：一步建站（域名 + 上游），站点能力三态识别与定点写回，复杂站点原文只读但防护可调
- **Caddy 配置**：预览、热加载、历史版本与回滚、全局选项/入口块/日志 snippet 单例配置
- **站点日志**：每站点开关访问/错误日志、单文件滚动上限、采集入库后按 `level`/`logger` 分流查询
- **证书中心**：Caddy 自动签发 + 用户上传双源合并、到期告警、缺口视图
- **IP 区域访问控制**：基于 ip2region 的国家/省份粒度限制
- **WAF**：Coraza + OWASP CRS，运行模式/强度/审计三档、CRS 规则在线更新与回滚、拦截事件查询
- **通知**：渠道、规则、模板、站内通知、发送日志
- **定时任务**：Cron 管理、脚本文件、执行日志
- **用户**：登录、冻结/解冻、首次启动引导建管理员。**无角色、无权限码、无动态菜单**

## 2. 技术栈

### 2.1 后端

| 项 | 值 |
|---|---|
| 目录 | `backend/` |
| 框架 | go-zero `v1.9.4` |
| Go | `1.27.0` |
| 数据库 | PostgreSQL + GORM |
| 缓存 | Redis（**可选**，连接失败不中断启动） |
| 认证 | JWT（`userId` + `tokenType`） |
| 日志 | go-zero `logx` + `internal/utils/logger` + 入库异步写入 |
| 后台 | 日志摄入、日志清理、证书同步、Cron 调度、WAF 调度、通知管理器 |

### 2.2 前端

| 项 | 值 |
|---|---|
| 目录 | `frontend/`（pnpm + Turborepo monorepo） |
| 基底 | vue-vben-admin v5 |
| 技术 | Vue 3.5、Vite 8、TypeScript 6、antdv-next（Ant Design Vue）、Pinia 3、Vue Router 5、Tailwind 4、Vue Query |
| 应用代码 | `frontend/apps/src/` |
| 共享包 | `frontend/packages/`（stores、hooks、utils、styles、types 等） |
| 路由 | **静态定义**，`frontend/apps/src/router/routes/modules/`，无后端下发菜单 |
| 请求层 | `frontend/apps/src/api/request.ts` |

> 历史遗留：早期基于 Soybean Admin + Naive UI + 动态路由。**这些描述已全部过时**，若在文档或注释里看到 Soybean / Naive UI / `frontend/src/` / `service/request` / 后端 `/api/route/*`，一律按本节纠正。

### 2.3 部署

- 单容器：前端静态资源 + 后端二进制 + 定制 Caddy（含 Coraza/CRS 模块）
- 默认入口 `http://localhost`（`LOGFLUX_HTTP_PORT` 可调）
- 链路：`Client → Caddy(:80) → 前端静态资源 / 后端(:8888) → PostgreSQL / Redis(可选)`
- `/api/health` 由后端应答，数据库探测失败返回 503
- WAF 工作目录固定 `/config/security`；证书上传目录 `/config/certs`

## 3. 仓库结构

```text
LogFlux/
├── backend/                 # go-zero 后端
│   ├── api/                 # .api 定义（唯一接口真相源）
│   ├── common/              # 公共能力：cron、gorm、ingest、logging、password、redis、result
│   ├── etc/                 # 本地配置（gitignore，仓库只有模板）
│   ├── internal/
│   │   ├── config/          # 配置结构体
│   │   ├── handler/         # goctl 生成，薄封装
│   │   ├── logic/           # goctl 生成 + 轻量编排
│   │   ├── middleware/      # 登录态、限流、IP 区域、响应包装
│   │   ├── notification/    # 通知系统（providers/ 下 7 种渠道）
│   │   ├── response/        # 统一响应结构
│   │   ├── service/         # 业务服务层
│   │   ├── svc/             # ServiceContext 依赖注入 + AutoMigrate
│   │   ├── tasks/           # 调度器与后台任务
│   │   ├── types/           # goctl 生成，禁止手改
│   │   ├── utils/           # logger、safego、timeparse
│   │   ├── waf/             # WAF 包获取、校验、解压、激活、CRS 布局
│   │   └── xerr/            # 统一错误码
│   ├── model/               # GORM 模型与 Model 接口（caddy/common/cron/ingest/notification/system/user/waf）
│   └── scripts/             # gen-api.sh、download-xdb.sh
├── frontend/                # vue-vben-admin 前端
│   └── apps/src/            # 业务代码（api、router、store、views、locales）
├── docker/                  # Dockerfile、Compose、运行时 Caddyfile、配置模板
├── docs/                    # 当前文档（caddy-architecture、waf、telegram）
├── contexts/context.md      # 本文件
└── Makefile                 # 常用部署命令
```

## 4. 后端架构规则

### 4.1 固定请求链路

```text
api/*.api
  -> goctl 生成 internal/types/types.go 与 internal/handler/routes.go
  -> internal/handler/*
  -> internal/logic/*
  -> internal/service/* 或 internal/logic/caddy/*
  -> model / internal/waf / internal/notification / common/ingest / internal/tasks
  -> PostgreSQL / Redis / Caddy Admin API / 外部服务
```

### 4.2 分层职责

- `.api` 是接口唯一真相源；改接口**必须先改** `backend/api/*.api`
- `internal/types/types.go` 与 `internal/handler/routes.go` 是生成文件，**禁止手改**
- `handler` 只做 `httpx.Parse` → 调 Logic → `result.HttpResult`
- `logic` 保持薄层，做校验与编排
- **Caddy/WAF 域例外**：`internal/logic/caddy/` 承担大量业务逻辑（Caddyfile 手术、策略渲染、WAF 激活），不再额外下沉 `service`——该域没有 `internal/service/caddy`。这是历史形成且当前有效的约定
- `model` 只做 GORM 数据访问，方法接收 `context.Context` 并用 `db.WithContext(ctx)`
- 长生命周期能力放 `internal/tasks`、`internal/notification`、`common/ingest`、`internal/waf`
- 基础设施依赖统一从 `svc.ServiceContext` 获取，**禁止**在业务层自建 DB、Redis 或第二套 logger

### 4.3 ServiceContext 当前字段

`backend/internal/svc/service_context.go`：

| 分组 | 字段 |
|---|---|
| 配置与基础设施 | `Config`、`DB`、`Redis` |
| 后台任务 | `LogCleanupTask`、`CronScheduler`、`WafScheduler`、`CertSyncScheduler`、`IngestMgr`、`NotificationMgr` |
| 中间件 | `ActiveUser`、`RateLimit`、`IPRegionCheck`、`IPRegionMgr` |
| Model | `SystemConfigModel`、`UserModel`、`CronTaskModel`、`CronTaskFileModel`、`CaddyLogModel`、`CertificateModel`、`SystemLogModel`、`WafEventModel` |

启动流程：`AutoMigrate` → 破坏性列清理（如 `notification_rules.event_type`）→ 单例配置迁移/补齐 → 默认账号、默认 WAF 源、默认策略播种 → 注册日志目录源 → 启动各调度器。

## 5. API 与 goctl 规范

### 5.1 API 文件

主入口 `backend/api/logflux.api`，当前 import：

```
base / auth / user / dashboard / caddy_server / caddy_certificate
caddy_global_options / caddy_ip_region / caddy_log / waf / notification / system_log / cron
```

所有业务路由前缀 `/api`，除登录/刷新/初始化外统一：

```api
@server (
    prefix: /api
    group:  xxx
    jwt:    Auth
    middleware: ActiveUser
)
```

`/api/health`、`/api/internal/geo-check` 在 `logflux.go` 直接注册（不走 goctl 分组），供 Caddy 健康检查与 `forward_auth` 调用。

### 5.2 生成命令

优先用脚本（会处理 goctl 脚手架噪声）：

```bash
cd backend
bash scripts/gen-api.sh
```

等价手动命令：

```bash
goctl api go -api api/logflux.api -dir . --style go_zero
```

- **必须** `--style go_zero`，保持 snake_case 文件名（默认 `gozero` 风格生成 camelCase，会与既有文件重复声明）
- 脚本会删掉重复的 `activeuser_middleware.go`、还原 `routes.go`、固定 `types.go` 版本注释
- 生成后只补齐业务实现，不手改生成文件

### 5.3 命名

| 对象 | 规则 | 示例 |
|---|---|---|
| 文件名 | snake_case | `get_user_list_logic.go` |
| 类型名 | PascalCase | `GetUserListLogic` |
| JSON/Form 字段 | 小驼峰 | `pageSize` |
| 请求/响应 | `Req` / `Resp` 后缀 | `CaddyLogReq` |

标签：查询参数 `form:"xxx"`、路径参数 `path:"id"`、Body `json:"xxx"`。保留 `.api` 的 `optional`、`default`、`range`、`options`。

## 6. 统一响应与错误

### 6.1 响应结构

`backend/internal/response/result.go`：

```json
{ "code": 0, "message": "成功", "msg": "成功", "data": {} }
```

- 成功码 `xerr.OK = 0`；`message` 是推荐字段，`msg` 仅供旧前端兼容
- 前端 `.env` 的 `VITE_SERVICE_SUCCESS_CODE=0`，与实际一致
- HTTP 状态恒为 200，业务状态由 `code` 表达

### 6.2 Handler 模板

```go
if err := httpx.Parse(r, &req); err != nil {
    httpx.ErrorCtx(r.Context(), w, err)
    return
}
l := xxx.NewXxxLogic(r.Context(), svcCtx)
resp, err := l.Xxx(&req)
result.HttpResult(r, w, resp, err)
```

`result.HttpResult`（`backend/common/result/http_result.go`）：成功走 `response.Success`，失败记中文日志后交 `httpx.ErrorCtx`。全局 `httpx.SetErrorHandler` 统一转成 `response.Result`。

### 6.3 错误体系

`backend/internal/xerr/xerr.go`：`OK=0`、`BusinessCommonError=400`、`Unauthorized=401`、`Forbidden=403`、`NotFound=404`、`ServerCommonError=500`。

| 场景 | 用法 |
|---|---|
| 业务校验 | `xerr.NewBusinessErrorWith("中文错误信息")` |
| 枚举类 | `xerr.NewEnumError(xerr.XXX)` |
| 需展示的系统错误 | `xerr.NewSystemErrorWith("中文错误信息")` |
| 需保留根因 | `xerr.NewCodeErrorWithCause(code, "中文错误信息", err)` |
| model 层 | 返回原始错误，上层用 `errors.Is(err, gorm.ErrRecordNotFound)` 判断 |

## 7. 数据库与模型

- PostgreSQL + GORM，模型在 `backend/model/<域>/`
- 禁止在 Handler 直接操作数据库；复杂查询沉到 model 方法
- Model 形态：`XxxModel interface` + `defaultXxxModel` + `NewXxxModel(db *gorm.DB) XxxModel`
- 默认不做软删除，状态用 `Status` 等业务字段表达
- 结构变更靠 `ServiceContext` 的 `AutoMigrate`；**破坏性变更必须写清迁移顺序**，参考 `migrateCaddyConfigToSystemConfig` 的幂等写法
- 单例配置统一存 `system_configs`（键值 + JSON），不新建单行表

## 8. 并发、任务与日志

### 8.1 Context

- Handler 用 `r.Context()`，全链路透传
- 只有启动流程、定时任务、消费者、后台异步才用 `context.Background()`

### 8.2 Goroutine

- 后台 goroutine 用 `internal/utils/safego`：`safego.New(ctx, "中文任务名").Go(func(){...})`，统一 recover 并记中文 panic 日志
- 定时任务复用 `internal/tasks` 的调度器，**不要另造线程池**
- Caddy 配置写入有全局互斥锁 `caddyConfigWriteMu`，新增写路径必须复用，不要开第二把锁

### 8.3 日志

- 业务代码用 `internal/utils/logger`，模块常量：`ModuleSystem`、`ModuleCaddy`、`ModuleLog`、`ModuleCron`、`ModuleNotification`、`ModuleUser`
- 日志文案中文，带关键业务上下文
- 禁止 `fmt.Print` 和标准库 `log` 输出运行日志

## 9. 认证与路由

**无 RBAC**。只有「是否登录」与「账号是否启用」两个判断。

| 能力 | 实现 |
|---|---|
| 签发 | `internal/service/auth_service.go`（JWT：`userId` + `tokenType`） |
| 登录态校验 | `internal/middleware/active_user_middleware.go`，检查用户存在且 `status=1` |
| 前端路由 | `frontend/apps/src/router/routes/modules/` 静态定义 |
| 初始化 | 库中无账号时前端引导到 `/auth/setup`（`GET /api/setup/status` + `POST /api/setup`），创建成功后入口自动关闭 |
| 密码规则 | `backend/common/password`（8–72 位且含字母与数字），HTTP 与 CLI 共用 |
| 忘记密码 | `logflux-api reset-password -f <config> -u <username>`，生成随机密码打印一次并解除禁用 |

历史 RBAC 代码（`permission_middleware.go`、role/menu/route 的 handler/logic/model/api）已移除，**不要重新引入**。

## 10. 前端开发规则

### 10.1 请求与响应

`frontend/apps/src/api/request.ts`：

- 响应 `transform` 已返回 `response.data.data`
- 业务代码 **不要再写** `res.data.xxx`
- 错误文案统一用 `frontend/apps/src/utils/api-error-message.ts` 的 `apiErrorMessage`，不要各处手写解析
- 错误 toast 至多一次；需要自行处理的场景用 suppress-global-toast

### 10.2 API 模块

`frontend/apps/src/api/` 按域拆分：

| 目录/文件 | 内容 |
|---|---|
| `core/` | `auth.ts`、`user.ts` |
| `caddy/` | `server.ts`、`certificate.ts`、`global-options.ts`、`ip-region.ts`、`log-retention.ts`、`log-snippet.ts`、`simple-waf.ts`、`source.ts`、`waf-event.ts` |
| `system/` | `log.ts`、`user.ts` |
| 根级 | `cron.ts`、`dashboard.ts`、`notification.ts`、`query-keys.ts`、`list-detail.ts` |

- 列表响应结构不统一时用 `_utils.ts` 的 `listOf` 归一
- 新增接口必须与 `.api` 的路径、方法、字段对齐
- 远程数据优先 Vue Query；query key 集中在 `query-keys.ts`

### 10.3 路由与页面

- 业务页面在 `frontend/apps/src/views/`：`caddy/`、`cron/`、`dashboard/`、`logs/`、`notification/`、`system/`、`user/`、`_core/`
- 路由文件在 `router/routes/modules/`，`hideInMenu: true` 表示路由保留但不出现在菜单
- 用户可见文案**必须中文化**
- 表格/表单/弹窗/按钮必须处理 loading、空数据、错误态
- 巨型页面拆分：纯逻辑抽到 `*-utils.ts` 或 `composables/`，并配单测

## 11. Caddy 与 WAF 开发红线

详细实现见 [`../docs/caddy-architecture.md`](../docs/caddy-architecture.md) 与 [`../docs/waf.md`](../docs/waf.md)。硬约束：

1. **Caddyfile 文本是唯一真相源**，结构化模型只用于展示
2. 站点改动**只能定点替换**，不做整体重渲染；无法识别的块原样保留
3. Caddy 配置写入必经 `caddyConfigApplyService.Apply`（adapt → load → persist），带 `baseHash` 乐观并发检查
4. 校验失败时运行态滚动回退、库不变，两侧都要有测试
5. **不要重新引入「节点」概念**：单实例，无 `serverId`，无节点选择器，无 remote
6. CRS 策略的 PL 必须用 `tx.blocking_paranoia_level` / `tx.detection_paranoia_level`，**不能**用裸 `tx.paranoia_level`
7. **不要**给 `/api/` 或 Host 为 IP 的请求注入 `ctl:ruleEngine=Off`（历史绕过漏洞，已修）
8. WAF 工作目录固定 `/config/security`，不要改到别处
9. Coraza 引擎不做在线替换，只做版本检查
10. `PUT /caddy/waf/simple-config` 只落库；`POST .../apply` 才改运行态

## 12. 开发工作流

### 12.1 改后端 API

1. 改 `backend/api/*.api`
2. `bash scripts/gen-api.sh`
3. 确认生成文件是 snake_case
4. Handler 保持薄封装用 `result.HttpResult`
5. 业务写 `internal/logic/<域>/`，DB 经 `svcCtx.XxxModel`
6. 错误统一走 `xerr`
7. 最小验证优先编译级：`go build ./... && go vet ./...`

### 12.2 改前端

1. 确认后端真实 `data` 结构
2. 更新 `frontend/apps/src/api/` 对应接口
3. 页面只用 `request` 返回的 data
4. 沿用现有组件与状态管理风格
5. 用户可见文本全中文
6. 补齐 loading / 空态 / 错误处理
7. 验证：`cd frontend && pnpm -F logflux-web run typecheck`

### 12.3 改数据库

1. 改 `backend/model/<域>/`
2. 更新 `ServiceContext` 的 `AutoMigrate`
3. 破坏性变更（删列、改名、加唯一索引）写清迁移顺序与幂等条件
4. 确认 Model 接口与调用方闭合

## 13. 常用命令

```bash
# 后端本地
cd backend && go mod download && bash scripts/download-xdb.sh
go run logflux.go -f etc/config.yaml

# 前端本地
cd frontend && pnpm install && pnpm run dev:antd

# 构建与部署
make download-xdb      # 下载 ip2region xdb
make build-backend     # 带 embed_ipregion 标签构建
make build-frontend
make up / down / restart / logs / status
```

`ip2region` 的 `.xdb` 不提交仓库。首次启动或编译前先 `bash backend/scripts/download-xdb.sh`（或 `make download-xdb`）。

## 14. 高风险约束

- **不要**把旧文档的 `code=200/msg=success` 当标准——成功码是 `0`
- **不要**绕开 `result.HttpResult` / `response.Result` / `xerr` 造第二套响应结构
- **不要**手改 `types.go`、`routes.go`
- **不要**在 Handler 写业务或直接操作 DB
- **不要**假设 Redis 必然可用
- **不要**把前端 `request` 返回值再包 `.data`
- **不要**覆盖用户在数据库里配置过的通知渠道、定时任务等运行时数据
- **不要**重新引入 `/security` 页面、策略版本、规则例外、策略绑定、误报反馈——这些能力与其接口已整体删除
