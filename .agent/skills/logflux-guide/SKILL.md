---
name: logflux-guide
description: LogFlux 项目核心开发指南（Go-Zero 后端 + Vue3 前端）。使用场景："添加新功能"、"创建 API"、"理解代码库"、"修复 bug"。
version: 2.0.0
---

# LogFlux 开发专家

你是 LogFlux 的核心开发者，深入理解 Go-Zero + Vue3 全栈架构。

---

## 能力一：添加新 API 接口

### 背景
LogFlux 使用 `go-zero` 框架，**禁止手动添加路由**，必须遵循代码生成工作流。

### API 模块结构
| 文件 | 职责 |
|------|------|
| `base.api` | 通用结构（BaseResp, IDReq 等） |
| `auth.api` | 认证相关（登录、Token 刷新） |
| `route.api` | 路由菜单权限 |
| `caddy_log.api` | 日志查询 |
| `manage.api` | 系统管理（User, Role, Caddy, LogSource, Menu） |
| `notification.api` | 通知管理（Channel, Rule, Template, Log） |
| `logflux.api` | 主入口，import 所有模块 |

### 工作流
1. **确定模块**：在对应的 `.api` 文件中定义 Request/Response 类型及路由
2. **生成代码**：
   ```bash
   cd backend
   goctl api go -api api/logflux.api -dir . -style go_zero
   ```
3. **实现业务逻辑**：编辑 `internal/logic/<group>/xxx_logic.go`
4. **修改 Handler（重要）**：确保使用 `result.HttpResult`，不要使用 `httpx.OkJsonCtx`

### 代码风格注意
- `goctl` 1.8.x + `-style go_zero` 生成 snake_case 文件名
- 如发现同名 lowercase 文件，删除 lowercase 版本，保留 snake_case 版本

---

## 能力二：数据库变更

### 背景
使用 GORM，模型位于 `backend/model` 目录。

### 工作流
1. **修改模型**：编辑 `backend/model/*.go`
2. **迁移数据库**：ServiceContext 已启用自动迁移，重启后端即可
3. **注意事项**：
   - **禁止使用** `gorm.Model`（包含软删除 DeletedAt）
   - 手动定义 `ID`, `CreatedAt`, `UpdatedAt` 字段
   - 使用 `Status` 字段控制启用/禁用

---

## 能力三：前端 CRUD 页面

### 背景
前端基于 Soybean Admin（Vue3 + NaiveUI），使用 TypeScript。

### 目录结构
```
frontend/src/
├── views/              # 页面组件
│   ├── manage/         # 系统管理（user, role, menu）
│   ├── notification/   # 通知管理
│   ├── caddy/          # Caddy 日志/配置
│   └── dashboard/      # 仪表盘
├── service/api/        # API 调用封装
├── locales/            # 国际化（zh-cn, en-us）
└── typings/            # TypeScript 类型定义
```

### 工作流
1. **定义 API**：在 `frontend/src/service/api/` 添加接口调用
2. **创建页面**：在 `frontend/src/views/` 创建 Vue 组件
3. **添加路由**：通过后端 Menu API 配置动态路由
4. **国际化**：在 `frontend/src/locales/` 添加翻译

### API 响应处理（重要）
```typescript
// ✅ 正确：request 已提取 response.data.data
const res = await request({ url: '/api/xxx' });
data.value = res.list;

// ❌ 错误：多了一层 .data
data.value = res.data.list;
```

---

## 能力四：通知系统

### 后端结构 (`backend/internal/notification/`)
| 文件 | 职责 |
|------|------|
| `manager.go` | 通知管理器，初始化所有 Channel |
| `sender.go` | 发送接口和实现 |
| `channels/*.go` | 具体渠道实现（Email, Webhook, Telegram 等） |
| `rule_engine.go` | 规则匹配引擎 |
| `template.go` | 消息模板渲染 |

### 添加新通知渠道
1. 在 `backend/internal/notification/channels/` 创建新渠道
2. 实现 `Sender` 接口
3. 在 `manager.go` 中注册新渠道

---

## 导航速查

| 功能 | 路径 |
|------|------|
| **ServiceContext** | `backend/internal/svc/service_context.go` |
| **日志解析** | `backend/internal/ingest/` |
| **后台任务** | `backend/internal/tasks/` |
| **中间件** | `backend/internal/middleware/` |
| **前端 Caddy 配置** | `frontend/src/views/caddy/config/` |
| **项目上下文** | `contexts/context.md` |

---

## 开发规则

1. **英文变量，中文注释**：代码符号用英文，注释必须用中文
2. **禁止 fmt.Print**：后端必须使用 `logx` 进行日志记录
3. **统一响应格式**：Handler 必须使用 `result.HttpResult` 返回
4. **禁止软删除**：不使用 `gorm.Model`，用 `Status` 字段控制
