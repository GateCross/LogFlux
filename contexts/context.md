# LogFlux 项目上下文

## 1. 项目定位

LogFlux 是一个日志流量分析、Caddy 图形化配置与 Coraza WAF 管理系统，采用前后端分离开发、一体化容器部署。

核心能力：

- Caddy 反向代理配置管理、预览、热加载、历史记录与回滚。
- Caddy 访问日志与运行日志采集、查询、统计与归档。
- Coraza + OWASP CRS 的简单防火墙配置、策略发布、版本管理、任务审计与误报反馈。
- RBAC 权限控制、动态菜单、用户/角色/菜单管理。
- 通知渠道、通知规则、模板、站内通知与通知日志。
- 定时任务管理与日志。

默认推荐入口：

- 常用 WAF 配置：`Caddy管理 -> Caddy配置 -> 防火墙`。
- 高级安全管理代码仍保留，默认入口已收敛，避免把简单 WAF 工作流拆散。

## 2. 技术栈

### 2.1 后端

- 目录：`backend/`
- 框架：go-zero `v1.9.4`
- Go 版本：`go 1.26.0`
- 数据库：PostgreSQL + GORM
- 缓存：Redis 可选
- 认证：JWT
- 日志：go-zero `logx` + `internal/utils/logger` + 数据库异步写入
- 后台能力：日志摄入、归档任务、Cron 调度、WAF 调度、通知管理器

### 2.2 前端

- 目录：`frontend/`
- 基底：Soybean Admin
- 技术：Vue 3、Vite 7、TypeScript、Naive UI、Pinia、UnoCSS、Elegant Router
- 请求：`frontend/src/service/request/index.ts`
- 路由模式：动态路由，后端通过 `/api/route/*` 提供用户路由

### 2.3 部署

- Docker 一体化容器包含：前端静态资源、后端二进制、Caddy。
- 默认入口：`http://localhost`
- 容器内链路：`Client -> Caddy(:80) -> Frontend + Backend(:8888) -> PostgreSQL / Redis`
- Docker Caddyfile 中 `/api/health` 由 Caddy 直接返回 `OK`。
- WAF 工作目录固定为 `/config/security`，包含 `packages` 与 `releases` 子目录。

## 3. 仓库结构

```text
LogFlux/
├── backend/                 # go-zero 后端
│   ├── api/                 # goctl API 定义
│   ├── common/              # 公共能力：GORM、Redis、响应、日志写入等
│   ├── etc/config.yaml      # 后端本地配置
│   ├── internal/
│   │   ├── config/          # 配置结构
│   │   ├── handler/         # goctl 生成 Handler，薄封装
│   │   ├── logic/           # goctl 生成 Logic，轻量编排
│   │   ├── middleware/      # 权限与响应中间件
│   │   ├── notification/    # 通知系统
│   │   ├── response/        # 统一响应结构
│   │   ├── service/         # 业务服务层
│   │   ├── svc/             # ServiceContext 依赖注入
│   │   ├── tasks/           # 后台任务与调度器
│   │   ├── types/           # goctl 生成类型，禁止手改
│   │   ├── utils/           # logger、safego、时间工具
│   │   ├── waf/             # WAF 包获取、验证、激活、存储
│   │   └── xerr/            # 统一错误码
│   └── model/               # GORM 模型与 Model 接口
├── frontend/                # Soybean Admin 前端
├── docker/                  # 镜像、Compose、运行时 Caddyfile 与配置模板
├── docs/                    # 设计、计划、部署与运维文档
├── contexts/context.md      # 当前项目核心上下文
└── Makefile                 # 常用部署命令
```

## 4. 后端架构规则

### 4.1 固定请求链路

```text
api/*.api
  -> goctl 生成 internal/types/types.go 与 internal/handler/routes.go
  -> internal/handler/*
  -> internal/logic/*
  -> internal/service/*
  -> model / internal/waf / internal/notification / internal/ingest / internal/tasks
  -> PostgreSQL / Redis / Caddy Admin API / 外部服务
```

### 4.2 分层职责

- `.api` 是接口唯一事实来源；新增或修改接口必须先改 `backend/api/*.api`。
- `internal/types/types.go` 与 `internal/handler/routes.go` 是生成文件，禁止手动修改。
- `handler` 只做 `httpx.Parse`、调用 logic、统一响应。
- `logic` 保持薄层，做轻量校验和编排；复杂业务下沉到 `service`。
- `service` 承载业务流程、缓存、锁、通知、任务、Caddy/WAF 编排。
- `model` 只做 GORM 数据访问；数据库操作优先经过 `Model interface`。
- 长生命周期能力放在 `internal/tasks`、`internal/notification`、`internal/ingest`、`internal/waf`。
- 基础设施依赖统一从 `svc.ServiceContext` 获取，禁止在业务层自行新建 DB、Redis 或第二套 logger。

### 4.3 ServiceContext 当前职责

`backend/internal/svc/service_context.go` 当前集中初始化：

- `Config`
- `DB`
- `Redis`
- `Ingestor`
- `ArchiveTask`
- `CronScheduler`
- `WafScheduler`
- `NotificationMgr`
- `Permission`
- `UserModel`、`RoleModel`、`MenuModel`
- `CronTaskModel`
- `LogSourceModel`
- `CaddyLogModel`
- `SystemLogModel`

启动时会执行 GORM `AutoMigrate`，初始化 RBAC、默认管理员、默认 WAF 源、默认 WAF 策略，并启动日志摄入、归档任务、Cron 调度和通知管理器。

## 5. API 与 goctl 规范

### 5.1 API 文件

主入口：`backend/api/logflux.api`

当前导入：

- `base.api`
- `auth.api`
- `route.api`
- `manage.api`
- `notification.api`
- `caddy_log.api`
- `system_log.api`
- `cron.api`

路由统一使用 `/api` 前缀。除登录、刷新 Token、常量路由等公开接口外，大部分接口使用：

```api
@server (
    prefix: /api
    group:  xxx
    jwt:    Auth
    middleware: Permission
)
```

### 5.2 生成命令

涉及 API 变更时执行：

```bash
cd backend
goctl api go -api api/logflux.api -dir . -style go_zero
```

约束：

- 必须使用 `-style go_zero`，保持 snake_case 文件名。
- 不要手改 `internal/types/types.go`。
- 不要手改 `internal/handler/routes.go`。
- goctl 生成后只补齐非生成业务实现。

### 5.3 命名与标签

- 文件名：snake_case，例如 `get_user_list_logic.go`。
- 类型名：PascalCase，例如 `GetUserListLogic`。
- JSON/Form 字段：小驼峰。
- 请求/响应类型优先使用 `Req` / `Resp` 后缀。
- 查询参数使用 `form:"xxx"`。
- 路径参数使用 `path:"id"`。
- JSON Body 使用 `json:"xxx"`。
- 保留 `.api` 中的 `optional`、`default`、`range`、`options` 等 DSL 标记。

## 6. 统一响应与错误

### 6.1 当前统一响应结构

后端统一响应定义在 `backend/internal/response/result.go`：

```json
{
  "code": 0,
  "message": "成功",
  "msg": "成功",
  "data": {}
}
```

说明：

- 当前业务成功码为 `xerr.OK = 0`。
- `message` 是当前推荐字段。
- `msg` 用于兼容旧前端。
- 前端请求层同时接受 `VITE_SERVICE_SUCCESS_CODE`、`0`、`200`，但新后端代码以 `0` 为准。

### 6.2 Handler 响应规则

推荐 Handler 模式：

```go
if err := httpx.Parse(r, &req); err != nil {
    httpx.ErrorCtx(r.Context(), w, err)
    return
}

l := xxx.NewXxxLogic(r.Context(), svcCtx)
resp, err := l.Xxx(&req)
result.HttpResult(r, w, resp, err)
```

`result.HttpResult` 位于 `backend/common/result/http_result.go`：

- 成功：`httpx.OkJsonCtx(r.Context(), w, response.Success(resp))`
- 失败：记录中文错误日志后交给 `httpx.ErrorCtx`

全局 `httpx.SetErrorHandler` 会把错误转换为 `response.ErrorFromErr(err)`，HTTP 状态仍返回 `200`，业务状态由 `code` 表达。

### 6.3 错误体系

错误码定义在 `backend/internal/xerr/xerr.go`：

- `xerr.OK = 0`
- `xerr.BusinessCommonError = 400`
- `xerr.Unauthorized = 401`
- `xerr.Forbidden = 403`
- `xerr.NotFound = 404`
- `xerr.ServerCommonError = 500`

使用规则：

- 业务校验错误：`xerr.NewBusinessErrorWith("中文错误信息")`
- 枚举类错误：`xerr.NewEnumError(xerr.XXX)`
- 需要展示给前端的系统错误：`xerr.NewSystemErrorWith("中文错误信息")`
- 需要保留根因：`xerr.NewCodeErrorWithCause(code, "中文错误信息", err)`
- `model` 层优先返回原始错误，上层用 `errors.Is(err, gorm.ErrRecordNotFound)` 判断。

## 7. 数据库与模型规范

- 使用 PostgreSQL + GORM。
- 模型位于 `backend/model/`。
- 禁止在 Handler 中直接操作数据库。
- 复杂查询和写入优先沉到 `model` 的接口方法。
- Model 形态优先保持：
  - `XxxModel interface`
  - `defaultXxxModel struct`
  - `NewXxxModel(db *gorm.DB) XxxModel`
  - 方法接收 `context.Context` 并使用 `db.WithContext(ctx)`
- 默认不使用软删除；用户、任务等状态通过 `Status` 或业务状态字段表达。
- 数据库结构启动时通过 `ServiceContext` 中的 `AutoMigrate` 同步，复杂迁移或破坏性变更必须写清楚迁移步骤。

## 8. 并发、任务与日志

### 8.1 Context

- Handler 使用 `r.Context()`。
- Logic、Service、Model 全链路透传 `context.Context`。
- 只有启动流程、定时任务、消费者、后台异步流程才使用 `context.Background()`。

### 8.2 Goroutine

- 新开后台 goroutine 优先使用 `internal/utils/safego`。
- `safego.New(ctx, "中文任务名").Go(func(){ ... })` 会统一 recover 并记录中文 panic 日志。
- 定时任务优先复用 `internal/tasks` 中的调度器，不要另造线程池。

### 8.3 日志

- 业务服务优先使用 `internal/utils/logger`。
- 常用模块包括：`ModuleSystem`、`ModuleCaddy`、`ModuleLog`、`ModuleCron`、`ModuleNotification`、`ModuleUser`。
- 日志文案使用中文，并带上关键业务上下文。
- 不要在业务代码中使用 `fmt.Print` 或标准库 `log` 输出运行日志。

## 9. RBAC 与路由

当前权限模型：

```text
User -> Roles ([]string) -> Permissions ([]string) -> Routes / Menus
```

默认角色：

- `admin`：管理员，拥有系统管理、Caddy、日志、安全、通知、个人中心等权限。
- `analyst`：分析师，可访问仪表盘、日志、Caddy 相关能力。
- `viewer`：访客，只读访问仪表盘。

后端关键位置：

- 权限中间件：`backend/internal/middleware/permission_middleware.go`
- 用户路由：`backend/internal/logic/route/get_user_routes_logic.go`
- RBAC 初始化：`backend/internal/svc/service_context.go`

菜单数据可能被用户在后台调整。初始化代码应尽量只补齐缺失数据或更新技术字段，避免覆盖用户自定义排序、图标、权限等配置。

## 10. 前端开发规则

### 10.1 请求响应

前端请求封装位于 `frontend/src/service/request/index.ts`。

当前 `transform` 已返回 `response.data.data`：

```ts
const res = await request({ url: '/api/xxx' });
// res 已经是后端 data 字段
```

因此业务代码不要再写 `res.data.xxx`。

错误文案优先读取 `message`，兼容 `msg`。

### 10.2 API 模块

前端 API 位于 `frontend/src/service/api/`，按后端领域拆分：

- `auth.ts`
- `dashboard.ts`
- `route.ts`
- `role.ts`
- `notification.ts`
- `cron.ts`
- `caddy*.ts`
- `system-log.ts`
- `log-source.ts`

新增接口时前后端路径、方法、请求字段必须与 `.api` 对齐。

### 10.3 路由与页面

- 业务页面位于 `frontend/src/views/`。
- 自动生成路由位于 `frontend/src/router/elegant/routes.ts`。
- 新页面通常需要配合 Soybean Admin 的路由生成流程，避免只手写生成文件。
- 用户可见文案必须中文化。
- 表格、表单、弹窗、按钮状态需要处理 loading、空数据与错误态。

## 11. WAF 与 Caddy 开发注意事项

- Caddy 配置热加载通过 Caddy Admin API 的 `/adapt` 与 `/load` 完成。
- 应用配置前必须先校验候选 Caddyfile，失败时不落库或明确回滚。
- 简单 WAF 配置接口前缀：`/api/caddy/waf/simple-config`。
- WAF 更新管理涉及：
  - 源：`WafSource`
  - Release：`WafRelease`
  - Job：`WafUpdateJob`
  - Policy：`WafPolicy`
  - Revision：`WafPolicyRevision`
  - Rule Exclusion：`WafRuleExclusion`
  - Binding：`WafPolicyBinding`
  - False Positive Feedback：`WafPolicyFalsePositiveFeedback`
- Coraza 引擎当前只做版本检查，不做在线替换。
- CRS 支持远端同步、上传、验证、激活、回滚。
- `Waf.WorkDir` 运行时会被收敛到 `/config/security`。

## 12. 开发工作流

### 12.1 新增或修改后端 API

1. 定位并修改 `backend/api/*.api`。
2. 执行 goctl 生成命令。
3. 检查生成的 Handler/Logic 文件名是否为 snake_case。
4. Handler 保持薄封装，使用 `result.HttpResult`。
5. Logic 做轻量编排。
6. 复杂业务写入 `internal/service/`。
7. DB 读写经 `model/` 与 `svcCtx.XxxModel`。
8. 错误统一走 `xerr`。
9. 最小验证优先做编译级检查；除非明确要求，不默认跑全量测试。

### 12.2 修改前端功能

1. 确认后端响应的真实 `data` 结构。
2. 更新 `frontend/src/service/api/` 中对应接口。
3. 页面只使用 `request` 返回的 data，不再访问额外 `.data`。
4. 保持 Soybean Admin 现有组件、路由和状态管理风格。
5. 用户可见文本全部中文化。
6. 补齐 loading、空数据、错误处理。

### 12.3 数据库变更

1. 修改 `backend/model/`。
2. 更新 `ServiceContext` 的 `AutoMigrate` 或明确迁移 SQL。
3. 对删除字段、重命名字段等破坏性变更写清迁移顺序。
4. 确认相关 Model 接口和 Service 调用闭合。

## 13. 常用命令

### 13.1 后端本地启动

```bash
cd backend
go mod download
go run logflux.go -f etc/config.yaml
```

### 13.2 前端本地启动

```bash
cd frontend
pnpm install
pnpm run dev
```

### 13.3 Docker 部署

```bash
cp docker/.env.example docker/.env
cp docker/config.example.yaml backend/etc/config.yaml
docker compose -f docker/docker-compose.yml up -d
```

也可以使用：

```bash
make up
make status
make logs
make restart
make down
```

## 14. 当前高风险约束

- 不要把旧文档中的 `code=200/msg=success` 当成后端新代码的唯一标准；当前后端标准成功码是 `0`，前端兼容 `200`。
- 不要绕开 `result.HttpResult`、`response.Result`、`xerr` 新增第二套响应结构。
- 不要手改 goctl 生成文件。
- 不要在 Handler 中写业务逻辑或数据库操作。
- 不要在 Logic 中堆大段复杂业务。
- 不要覆盖用户已在数据库里配置过的菜单、权限、通知渠道等运行时数据。
- 不要把 WAF 工作目录改到 `/config/security` 之外。
- 不要假设 Redis 必然可用；当前 Redis 连接失败不会中断启动。
- 不要把前端 `request` 返回值再包一层 `.data`。

