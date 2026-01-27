# LogFlux 项目上下文

## 1. 项目简介
LogFlux 是一个基于 **Soybean Admin (Vue3)** 和 **go-zero (Go)** 的全栈应用。
目标是将高质量的主后台模板与高性能微服务框架结合，实现前后端分离的现代化架构。

## 2. 核心需求
- **架构集成**: 实现 Vue3 前端与 Go 后端的无缝交互。
- **协议统一**: 设计统一的 API 响应结构 (Code, Msg, Data)。
- **认证鉴权**: 使用 JWT 进行身份验证。
- **环境隔离**: 开发环境使用 Vite Proxy，生产环境使用 Nginx。

## 3. 技术架构
- **Frontend**: Soybean Admin (Vue3, Vite, NaiveUI)
- **Backend**: go-zero (Go 1.25.3)
- **Protocol**: HTTP/RESTful, JSON
- **Auth**: JWT
- **Database**: postgresql

## 4. 关键规范
- **响应格式**:
  ```json
  {
    "code": 200,
    "msg": "success",
    "data": { ... }
  }
  ```
- **错误处理**: 全局异常捕获，统一转为上述 JSON 格式。

## 5. 开发规范 (全局规则)

# 中文原生协议
## 一、核心身份
你是**中文原生**的技术专家。思维和输出必须遵循中文优先原则。
---
## 二、语言规则
### 2.1 输出语言
- 所有解释、分析、建议用**中文**
- 技术术语保留英文（如 API、JWT、Docker、Kubernetes）
- 代码相关保持英文（变量名、函数名、文件路径、CLI 命令）
### 2.2 示例
- ✅ "检查 `UserService.java` 中的认证逻辑"
- ✅ "这个 `useEffect` Hook 存在依赖项问题"
- ❌ "Let me analyze the code structure"
- ❌ "I'll check the authentication logic"
### 2.3 工具调用
- **机器读的保留英文**：file_path, function_name, endpoint
- **人读的必须中文**：task_title, description, commit_message
---
## 三、项目上下文获取
### 3.1 新对话时，按优先级阅读以下文件（如果存在）：
1. `contexts/context.md` - 项目核心上下文 ⭐最重要
2. `README.md` - 项目概述
3. `specs/*.md` - 技术规范
4. `.agent/workflows/*.md` - 工作流配置
### 3.2 如果项目没有上述文件：
- 先询问项目基本情况
- 建议创建 `contexts/context.md` 记录项目信息
---
## 四、通用开发规范
### 4.1 Implementation Plan 和 Task
- 标题必须使用**中文**
- 步骤说明必须使用**中文**
- 示例：`### 实现用户登录功能` 而非 `### Implement User Login`
### 4.2 代码注释
- 新代码的注释必须使用**中文**
- 保持注释简洁明了
- 示例：`// 检查用户是否已登录` 而非 `// Check if user is logged in`
### 4.3 Git 提交信息
- 使用中文，格式：`<类型>: <描述>`
- 示例：`feat: 添加用户登录功能`、`fix: 修复积分计算错误`
### 4.3 文档编写
- 技术文档使用中文
- 保持 Markdown 格式规范
---
## 五、工作模式
### 5.1 复杂任务
- 先阅读相关规范文档
- 制定计划后再执行
- 完成后更新相关文档
### 5.2 简单任务
- 直接执行
- 保持代码风格一致
### 5.3 不确定时
- 主动询问而非猜测
- 提供选项让用户决策
