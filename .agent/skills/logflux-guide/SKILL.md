---
name: logflux-guide
description: LogFlux 项目核心开发指南（Go-Zero 后端 + Vue3 前端）。使用场景："添加新功能"、"创建 API"、"理解代码库"、"修复 bug"。
version: 3.0.0
---

# LogFlux 开发专家

你是 LogFlux 的核心开发者，深入理解 Go-Zero + Vue3 全栈架构。

---

## 能力一：Go-Zero API 开发

### API 文件结构

```
backend/api/
├── logflux.api                 # 主入口，import 所有模块
├── base.api                    # 通用结构（BaseResp, IDReq 等）
├── auth.api                    # 登录、刷新令牌、初始化管理员
├── user.api                    # 用户管理（无角色/权限）
├── dashboard.api               # 仪表盘统计
├── caddy_server.api            # Caddy 配置、历史、回滚、Docker 发现
├── caddy_certificate.api       # 证书中心
├── caddy_global_options.api    # 全局选项 / 入口块 / 日志 snippet / 保留策略
├── caddy_ip_region.api         # IP 区域访问控制
├── caddy_log.api               # 日志查询与站点指标
├── waf.api                     # WAF 源、上传、引擎、策略、事件
├── notification.api            # 通知渠道/规则/模板/日志
├── system_log.api              # 系统日志
└── cron.api                    # 定时任务
```

> 历史文件 `route.api` / `menu.api` / `role.api` / `manage.api` 已随 RBAC 与 `/security` 页面删除，不要重新创建。

### API 定义语法

```api
// 定义类型
type (
    // 请求结构
    CreateUserReq {
        Username string `json:"username"`           // 必填
        Email    string `json:"email,optional"`     // 可选
        Status   int    `json:"status,default=1"`   // 默认值
    }
    
    // 响应结构
    CreateUserResp {
        Id uint `json:"id"`
    }
)

// 定义路由组
@server (
    prefix: /api/user      // 路由前缀
    group:  user           // 分组名（生成目录）
    jwt:    Auth           // 启用 JWT 认证
    middleware: Permission // 中间件
)
service logflux-api {
    @doc "创建用户"
    @handler CreateUser
    post /create (CreateUserReq) returns (CreateUserResp)
    
    @doc "获取用户列表"
    @handler GetUserList
    get /list (PageReq) returns (UserListResp)
}
```

### 常用标签

| 标签 | 说明 | 示例 |
|------|------|------|
| `optional` | 可选字段 | `json:"name,optional"` |
| `default` | 默认值 | `json:"status,default=1"` |
| `options` | 枚举值 | `json:"type,options=email\|webhook\|telegram"` |
| `range` | 数值范围 | `json:"page,range=[1:]"` |

### 代码生成

```bash
cd backend

# 推荐：用脚本生成（会处理 goctl 脚手架噪声）
bash scripts/gen-api.sh

# 等价手动命令（使用 snake_case 风格）
goctl api go -api api/logflux.api -dir . --style go_zero

# 约束：必须使用 --style go_zero（下划线命名），禁止 gozero/goZero。
# 若出现 addcaddyserverhandler.go 等无下划线文件，先删除再重新生成。

# 生成后的目录结构
internal/
├── handler/
│   └── user/
│       ├── create_user_handler.go    # 自动生成
│       └── get_user_list_handler.go
├── logic/
│   └── user/
│       ├── create_user_logic.go      # 自动生成（需实现）
│       └── get_user_list_logic.go
└── types/
    └── types.go                       # 所有类型定义
```

### 关于 Handler 响应方式

**项目已实现 `ResponseMiddleware` 自动包装响应**，位于 `backend/internal/middleware/response_middleware.go`。

中间件会自动检测响应是否含 `code` 且含 `message` 或 `msg`：
- **未包装**：按 `xerr.OK = 0` 包装成 `{code: 0, message: "成功", msg: "成功", data: {...}}`
- **已包装**：直接放行
- **非 JSON 或非 200**：原样透传

**因此以下两种写法都可以正常工作**：

```go
// ✅ goctl 生成的默认代码（中间件会自动包装）
if err != nil {
    httpx.ErrorCtx(r.Context(), w, err)
} else {
    httpx.OkJsonCtx(r.Context(), w, resp)
}

// ✅ 显式使用 result.HttpResult（已包装，中间件跳过）
result.HttpResult(r, w, resp, err)
```

> 📌 **推荐**：新生成的 Handler 可保持默认代码，中间件会自动处理。使用 `result.HttpResult` 可以更明确地控制错误响应格式。

---

## 能力二：Go-Zero 中间件开发

### 中间件注册

```go
// backend/internal/middleware/目录下创建中间件

// logflux.go 中注册
server := rest.MustNewServer(c.RestConf,
    rest.WithUnauthorizedCallback(jwtUnauthorizedCallback),
    // 全局中间件
)
server.Use(middleware.NewResponseMiddleware().Handle)
```

### 登录态中间件（无 RBAC）

```go
// backend/internal/middleware/active_user_middleware.go
// 只校验「用户存在 + status=1」，不做角色/权限判断。
// 历史 PermissionMiddleware 已随 RBAC 删除，不要重新实现。
type ActiveUserMiddleware struct {
    db *gorm.DB
}

func (m *ActiveUserMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        userID, err := userIDFromContext(r)
        if err != nil {
            writeAuthError(w, http.StatusUnauthorized, 401, "登录状态无效")
            return
        }
        var user usermodel.User
        if err := m.db.Select("id", "status").First(&user, userID).Error; err != nil {
            // gorm.ErrRecordNotFound → 401「用户不存在」
            return
        }
        if user.Status == 0 {
            writeAuthError(w, http.StatusForbidden, 403, "用户已被禁用")
            return
        }
        next(w, r)
    }
}
```

### 响应包装中间件

```go
// 项目已实现的响应中间件
// backend/internal/middleware/response_middleware.go
type ResponseMiddleware struct{}

func (m *ResponseMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // 包装 ResponseWriter 以捕获响应
        rw := rwPool.Get().(*responseWriter)
        defer func() {
            rw.Reset()
            rwPool.Put(rw)
        }()
        rw.ResponseWriter = w
        
        next(rw, r)
        
        // 处理响应...
    }
}
```

---

## 能力三：Go-Zero 配置管理

### 配置文件结构

```yaml
# backend/etc/config.yaml
Name: logflux-api
Host: 0.0.0.0
Port: 8080

# JWT 配置
Auth:
  AccessSecret: your-secret-key
  AccessExpire: 604800  # 7 天（秒）

# 数据库配置
Database:
  Host: localhost
  Port: 5432
  User: postgres
  Password: password
  DBName: logflux
  SSLMode: disable

# Redis 配置（可选）
Redis:
  Host: localhost:6379
  Pass: ""
  DB: 0

# 通知配置
Notification:
  Enabled: true
  Channels:
    - name: "telegram"
      type: "telegram"
      enabled: true
      config:
        bot_token: "xxx"
        chat_id: "123"
```

### 配置结构体

```go
// backend/internal/config/config.go
package config

import "github.com/zeromicro/go-zero/rest"

type Config struct {
    rest.RestConf
    Auth         AuthConf
    Database     DatabaseConf
    Redis        RedisConf     `json:",optional"`
    Notification NotificationConf `json:",optional"`
}

type AuthConf struct {
    AccessSecret string
    AccessExpire int64
}

type DatabaseConf struct {
    Host     string
    Port     int
    User     string
    Password string
    DBName   string
    SSLMode  string `json:",default=disable"`
}
```

---

## 能力四：ServiceContext 模式

### 核心概念

ServiceContext 是 go-zero 的依赖注入容器，所有共享资源在此初始化：

```go
// backend/internal/svc/service_context.go
package svc

type ServiceContext struct {
    Config          config.Config
    DB              *gorm.DB
    Redis           *redis.Client
    NotificationMgr notification.NotificationManager
}

func NewServiceContext(c config.Config) *ServiceContext {
    svcCtx := &ServiceContext{
        Config: c,
    }
    
    // 初始化数据库
    svcCtx.DB = initDatabase(c.Database)
    
    // 自动迁移
    svcCtx.DB.AutoMigrate(
        &model.User{},
        &model.Role{},
        &model.NotificationChannel{},
    )
    
    // 初始化 Redis（可选）
    if c.Redis.Host != "" {
        svcCtx.Redis = initRedis(c.Redis)
    }
    
    // 初始化通知管理器
    svcCtx.NotificationMgr = initNotificationManager(svcCtx.DB, svcCtx.Redis)
    
    return svcCtx
}
```

### Logic 中使用

```go
// backend/internal/logic/user/get_user_logic.go
type GetUserLogic struct {
    logx.Logger
    ctx    context.Context
    svcCtx *svc.ServiceContext
}

func NewGetUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserLogic {
    return &GetUserLogic{
        Logger: logx.WithContext(ctx),
        ctx:    ctx,
        svcCtx: svcCtx,
    }
}

func (l *GetUserLogic) GetUser(req *types.GetUserReq) (*types.GetUserResp, error) {
    var user model.User
    
    // 使用 svcCtx.DB 查询
    if err := l.svcCtx.DB.First(&user, req.Id).Error; err != nil {
        l.Errorf("查询用户失败: %v", err)  // 使用内置 Logger
        return nil, errors.New("用户不存在")
    }
    
    return &types.GetUserResp{
        Id:       user.ID,
        Username: user.Username,
    }, nil
}
```

---

## 能力五：JWT 认证

### 配置 JWT

```yaml
# config.yaml
Auth:
  AccessSecret: your-256-bit-secret
  AccessExpire: 604800  # 7 天
```

### 生成 Token

```go
// backend/internal/logic/auth/login_logic.go
func (l *LoginLogic) Login(req *types.LoginReq) (*types.LoginResp, error) {
    // 验证用户...
    
    // 生成 Token
    now := time.Now().Unix()
    token, err := l.generateToken(user.ID, now)
    if err != nil {
        return nil, err
    }
    
    return &types.LoginResp{
        Token:  token,
        Expire: now + l.svcCtx.Config.Auth.AccessExpire,
    }, nil
}

func (l *LoginLogic) generateToken(userId uint, iat int64) (string, error) {
    claims := make(jwt.MapClaims)
    claims["userId"] = userId
    claims["exp"] = iat + l.svcCtx.Config.Auth.AccessExpire
    claims["iat"] = iat
    
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString([]byte(l.svcCtx.Config.Auth.AccessSecret))
}
```

### 获取用户信息

```go
// 在 Logic 中获取 JWT 中的 userId
func (l *SomeLogic) DoSomething(req *types.Req) error {
    userId := l.ctx.Value("userId").(json.Number)
    uid, _ := userId.Int64()
    
    // 使用 userId...
}
```

---

## 能力六：错误处理

### 统一错误响应

```go
// backend/common/result/http_result.go
package result

import (
    "net/http"
    "github.com/zeromicro/go-zero/rest/httpx"
)

type Response struct {
    Code int         `json:"code"`
    Msg  string      `json:"msg"`
    Data interface{} `json:"data,omitempty"`
}

func HttpResult(r *http.Request, w http.ResponseWriter, data interface{}, err error) {
    if err != nil {
        httpx.WriteJson(w, http.StatusOK, &Response{
            Code: 500,
            Msg:  err.Error(),
        })
        return
    }
    
    httpx.WriteJson(w, http.StatusOK, &Response{
        Code: 200,
        Msg:  "success",
        Data: data,
    })
}
```

### 自定义业务错误

```go
// backend/common/errors/errors.go
package errors

type BizError struct {
    Code int
    Msg  string
}

func (e *BizError) Error() string {
    return e.Msg
}

var (
    ErrUserNotFound    = &BizError{Code: 1001, Msg: "用户不存在"}
    ErrInvalidPassword = &BizError{Code: 1002, Msg: "密码错误"}
    ErrPermissionDeny  = &BizError{Code: 1003, Msg: "权限不足"}
)
```

---

## 能力七：数据库操作

### GORM 模型规范

```go
// backend/model/user.go
package model

import "time"

// ❌ 禁止使用 gorm.Model（包含软删除）
// ✅ 手动定义字段
type User struct {
    ID        uint      `gorm:"primarykey" json:"id"`
    CreatedAt time.Time `json:"createdAt"`
    UpdatedAt time.Time `json:"updatedAt"`
    Username  string    `gorm:"uniqueIndex;size:50;not null" json:"username"`
    Password  string    `gorm:"size:100;not null" json:"-"`
    Email     string    `gorm:"size:100" json:"email"`
    Status    int       `gorm:"default:1;not null" json:"status"`  // 1=启用, 0=禁用
    // 无 Roles 字段：RBAC 已删除，只有「是否登录」与「账号是否启用」两个判断
}
```

### 常用查询模式

```go
// 分页查询
func (l *Logic) GetList(req *types.PageReq) (*types.ListResp, error) {
    var list []model.User
    var total int64
    
    db := l.svcCtx.DB.Model(&model.User{})
    
    // 条件过滤
    if req.Username != "" {
        db = db.Where("username LIKE ?", "%"+req.Username+"%")
    }
    
    // 获取总数
    db.Count(&total)
    
    // 分页
    offset := (req.Page - 1) * req.PageSize
    if err := db.Offset(offset).Limit(req.PageSize).Find(&list).Error; err != nil {
        return nil, err
    }
    
    return &types.ListResp{
        List:  list,
        Total: total,
    }, nil
}
```

---

## 导航速查

| 功能 | 路径 |
|------|------|
| **API 定义** | `backend/api/*.api` |
| **Handler** | `backend/internal/handler/` |
| **Logic** | `backend/internal/logic/` |
| **类型定义** | `backend/internal/types/types.go` |
| **配置** | `backend/internal/config/config.go` |
| **ServiceContext** | `backend/internal/svc/service_context.go` |
| **中间件** | `backend/internal/middleware/` |
| **模型** | `backend/model/` |
| **统一响应** | `backend/common/result/` |
| **项目上下文** | `contexts/context.md` |

---

## goctl 常用命令

```bash
# API 代码生成（推荐脚本，会清理 goctl 脚手架噪声）
bash scripts/gen-api.sh

# 等价手动命令
goctl api go -api api/logflux.api -dir . --style go_zero

# 查看 API 文档
goctl api doc --dir api

# 格式化 API 文件
goctl api format --dir api

# 验证 API 文件
goctl api validate --api api/logflux.api
```

---

## 开发规则

1. **禁止手动添加路由** - 必须通过 `.api` 文件定义，使用 `goctl` 生成
2. **Handler 必须使用** `result.HttpResult` - 保证响应格式统一
3. **禁止使用** `gorm.Model` - 手动定义字段，不使用软删除
4. **日志使用** `logx` - 禁止 `fmt.Print`
5. **注释使用中文** - 保持代码可读性
