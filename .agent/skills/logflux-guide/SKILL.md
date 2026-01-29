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
1.  **定位 API 定义**: 在 `backend/api` (或根目录) 找到相关的 `.api` 文件。
2.  **定义 Handler/Logic**: 确定所属的分组 (Group) (例如: `server`, `user`)。
3.  **实现业务逻辑**:
    - **不要** 手动修改 `handler/*` 下的文件。
    - 专注于编辑 `internal/logic/<group>/<action>_logic.go`。
    - 通过 `ServiceContext` (`svcCtx`) 注入依赖。

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
