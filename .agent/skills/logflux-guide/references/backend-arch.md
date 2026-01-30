# LogFlux 后端架构参考

LogFlux 使用 [Go-Zero](https://go-zero.dev/) 框架。

## 目录结构 (`backend/`)

```
backend/
├── api/                    # API 定义（.api 文件）
│   ├── logflux.api         # 主入口，import 所有模块
│   ├── base.api            # 通用结构
│   ├── auth.api            # 认证相关
│   ├── route.api           # 路由菜单
│   ├── manage.api          # 系统管理
│   ├── notification.api    # 通知管理
│   └── caddy_log.api       # 日志查询
├── internal/
│   ├── config/             # 配置结构定义
│   ├── svc/                # ServiceContext（依赖注入）
│   ├── handler/            # HTTP Handler（禁止放业务逻辑）
│   ├── logic/              # 业务逻辑层
│   ├── types/              # Request/Response 结构
│   ├── middleware/         # 中间件
│   ├── ingest/             # 日志摄入解析
│   ├── notification/       # 通知系统
│   └── tasks/              # 后台任务
├── model/                  # 数据库模型（GORM）
├── common/                 # 公共工具
│   └── result/             # 统一响应格式
└── etc/                    # 配置文件
    └── logflux.yaml
```

## 核心组件

### ServiceContext (`internal/svc/service_context.go`)
- 全局依赖注入容器
- 初始化：DB、Redis、Model、后台任务

### 统一响应 (`common/result/`)
- 所有 Handler 必须使用 `result.HttpResult(r, w, resp, err)`
- 禁止直接使用 `httpx.OkJsonCtx`

### 日志摄入 (`internal/ingest/`)
- 解析 Caddy 日志
- 文件流读取

### 通知系统 (`internal/notification/`)
- `manager.go` - 管理器
- `sender.go` - 发送接口
- `channels/` - 渠道实现
- `rule_engine.go` - 规则匹配

## 开发流程

1. **定义 API**：修改 `api/*.api` 文件
2. **生成代码**：
   ```bash
   cd backend
   goctl api go -api api/logflux.api -dir . -style go_zero
   ```
3. **实现逻辑**：编辑 `internal/logic/`
4. **注册依赖**：必要时修改 `internal/svc/service_context.go`
