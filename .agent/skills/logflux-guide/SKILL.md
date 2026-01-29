---
name: logflux-guide
description: Expert knowledge about LogFlux project structure (Go-Zero backend + Vue3 frontend). Use when asked to "implement a feature", "add an API", or "understand the codebase".
version: 1.0.0
---

# LogFlux 专家 (LogFlux Specialist)

你是 LogFlux 的核心开发者。你深入理解全栈架构。

## 能力：添加新的 API 接口 (Add New API Endpoint)

### Context (背景)
LogFlux 使用 `go-zero` 框架。你不能手动直接添加路由，必须遵循代码生成的工作流。

### Workflow (工作流)
1.  **定位 API 定义**: API 定义已拆分为多个模块：
    - `base.api`: 通用结构 (BaseResp, IDReq)
    - `auth.api`: 认证相关
    - `route.api`: 路由菜单
    - `caddy_log.api`: 日志查询
    - `manage.api`: 系统管理 (User, Role, Caddy, LogSource)
    - `notification.api`: 通知管理 (Channel, Rule, Template, Log)
    - `logflux.api`: 主入口，负责 import 所有模块。
2.  **定义 Handler/Logic**: 确定所属的分组 (Group) 在哪个 API 文件定义。
3.  **实现业务逻辑**:
    - **注意**: `goctl` 1.8.x + `-style go_zero` 会生成 snake_case 文件名。如果发现 logic/handler 目录下有同名但 lowercase 的文件，请删除 lowercase 版本，保留 snake_case 版本并确保逻辑迁移。

## 能力：数据库变更 (Database Changes)

### Context (背景)
我们使用 GORM 或 SQLX (检查 `model` 目录)。

### Workflow (工作流)
1.  **修改模型**: 编辑 `backend/model` 下的结构体。
2.  **迁移 (Migration)**: 确保数据库 Schema 更新 (检查 `ServiceContext` 是否开启了自动迁移，或通过 SQL 变更)。

## 导航提示 (Navigation Tips)
- **Frontend Config**: `frontend/src/views/caddy/config`
- **Backend Service Context**: `backend/internal/svc/service_context.go` 是连接所有依赖的“胶水”层。
- **Log Parsing**: 查看 `backend/internal/ingest`。

## 规则 (Rules)
- **英文变量，中文注释**: 严格遵守项目规范，代码符号用英文，但**注释必须用中文**。
- **禁止直接 fmt.Print**: 后端必须使用 `logx` 进行日志记录。
