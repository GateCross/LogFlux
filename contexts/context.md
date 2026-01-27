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

---

## 六、后端开发标准

### 6.1 API 响应格式（强制）
**所有 Handler 必须使用 `result.HttpResult` 统一返回格式**：

```go
import "logflux/common/result"

func SomeHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // ... 解析请求
        
        l := logic.NewSomeLogic(r.Context(), svcCtx)
        resp, err := l.SomeMethod(&req)
        result.HttpResult(r, w, resp, err)  // ✅ 正确
    }
}
```

❌ **禁止直接使用** `httpx.OkJsonCtx`（会导致前端无法正确解析）：
```go
httpx.OkJsonCtx(r.Context(), w, resp)  // ❌ 错误：缺少 code/msg 包装
```

**标准响应结构**：
```json
{
  "code": 200,
  "msg": "success",
  "data": { ... }
}
```

### 6.2 数据库设计标准

#### 6.2.1 禁用软删除
- **不使用** `gorm.Model`（包含 DeletedAt）
- **手动定义字段**，移除 `DeletedAt`

```go
// ✅ 正确
type User struct {
    ID        uint      `gorm:"primarykey"`
    CreatedAt time.Time
    UpdatedAt time.Time
    Username  string    `gorm:"uniqueIndex;not null"`
    // ...
}

// ❌ 错误
type User struct {
    gorm.Model  // 包含不需要的 DeletedAt
    Username string
}
```

#### 6.2.2 状态管理
使用 `Status` 字段控制启用/禁用，而非软删除：

```go
type User struct {
    // ...
    Status int `gorm:"default:1;not null"` // 1=启用, 0=禁用
}
```

#### 6.2.3 数据库迁移
- 字段添加使用 `ALTER TABLE ... ADD COLUMN IF NOT EXISTS`
- 字段删除使用 `ALTER TABLE ... DROP COLUMN IF EXISTS`
- 先执行数据库迁移，再重启后端

---

## 七、RBAC 权限系统

### 7.1 权限模型
```
User → Roles ([]string) → Permissions ([]string) → Routes
```

### 7.2 父级权限自动授予
- 选择子权限时，自动授予父权限
- 例：选择 `manage_user` 时，自动授予 `manage`

**实现位置**：`backend/internal/logic/route/get_user_routes_logic.go`

```go
// 示例：自动添加父级权限
if hasChildPermission {
    permissions["parent_permission"] = true
}
```

### 7.3 前端权限选项
- 按模块分组：Dashboard、日志管理、系统管理
- 隐藏父级权限选项（自动授予）
- 避免冗余选项

---

## 八、前端开发标准

### 8.1 表格和列表

#### 8.1.1 中文化
所有面向用户的文本必须使用中文：

```typescript
// ✅ 正确
const columns = [
  { title: '时间', key: 'time' },
  { title: '状态', key: 'status' }
];

// ❌ 错误
const columns = [
  { title: 'Time', key: 'time' },
  { title: 'Status', key: 'status' }
];
```

#### 8.1.2 角色/状态显示
使用映射表显示中文：

```typescript
const roleMap: Record<string, string> = {
  admin: '管理员',
  analyst: '分析师',
  viewer: '访客'
};
```

### 8.2 API 请求处理

#### 8.2.1 响应数据访问
`request` 函数的 `transform` 已提取 `response.data.data`：

```typescript
// ✅ 正确
const res = await request({ url: '/api/xxx' });
data.value = res.list;  // res 已经是 data 对象

// ❌ 错误
data.value = res.data.list;  // 多了一层 .data
```

#### 8.2.2 错误处理
必须正确处理空数据和错误：

```typescript
async function fetchData() {
  loading.value = true;
  try {
    const res = await request<any>({ url: '/api/xxx' });
    if (res) {  // 检查响应存在
      data.value = res.list || [];
      total.value = res.total || 0;
    }
  } catch (error) {
    console.error('获取数据失败:', error);
    data.value = [];
    total.value = 0;
  } finally {
    loading.value = false;
  }
}
```

---

## 九、常见问题和解决方案

### 9.1 "backend request error"
**原因**：Handler 未使用 `result.HttpResult`
**解决**：修改 Handler 使用统一响应格式

### 9.2 数据库查询错误（deleted_at 不存在）
**原因**：模型移除了软删除但数据库还有 deleted_at 列
**解决**：执行数据库迁移删除该列

### 9.3 前端数据显示为空
**原因**：错误地访问 `res.data.xxx`，应该是 `res.xxx`
**解决**：理解 `transform` 函数已提取数据层级

### 9.4 权限 403 错误
**原因**：父权限未自动授予
**解决**：实现父权限自动授予逻辑

---

## 十、开发流程规范

### 10.1 新增 API
1. 定义 types（Request/Response）
2. 实现 Logic
3. 创建 Handler（**必须使用** `result.HttpResult`）
4. 注册路由
5. 测试 API 返回格式

### 10.2 数据库变更
1. 修改 Model
2. 编写迁移 SQL
3. 执行数据库迁移
4. 重启后端服务
5. 验证功能

### 10.3 前端开发
1. 定义 TypeScript 接口
2. 实现 API 调用函数
3. 正确访问响应数据（注意 transform）
4. 添加错误处理
5. 中文化所有用户界面文本
