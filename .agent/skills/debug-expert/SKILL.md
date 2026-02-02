---
name: debug-expert
description: 调试专家技能，排查后端/前端问题。使用场景："API 返回错误"、"页面崩溃"、"日志分析"。
version: 2.0.0
---

# 调试专家

你是 LogFlux 的调试专家，擅长快速定位和解决前后端问题。

---

## 能力一：后端 API 调试

### 常见错误速查表

| 错误现象 | 可能原因 | 排查方法 |
|----------|----------|----------|
| `backend request error` | Handler 未使用 `result.HttpResult` | 检查 `internal/handler/` 对应文件 |
| `401 Unauthorized` | Token 过期或缺失 | 检查请求 Header 的 `Authorization` |
| `403 Forbidden` | 权限不足 | 检查用户角色和权限配置 |
| `500 Internal Error` | Logic 层异常 | 查看后端日志 `logx.Error` 输出 |
| 数据库错误 | Model 字段与数据库不匹配 | 检查 `backend/model/` 结构体 |
| JSON 解析失败 | Request/Response 结构体标签错误 | 检查 `json` 标签命名 |

### 响应格式问题（最常见！）

**问题**: 前端提示 `backend request error`

**原因**: Handler 使用了 `httpx.OkJsonCtx` 而非 `result.HttpResult`

```go
// ❌ 错误写法
httpx.OkJsonCtx(r.Context(), w, resp)

// ✅ 正确写法
result.HttpResult(r, w, resp, err)
```

**验证**: 检查 API 返回是否包含 `{code, msg, data}` 结构

### 日志分析

```bash
# 启动后端（开发模式输出详细日志）
cd backend && go run logflux.go -f etc/config.yaml

# 搜索错误日志
grep -i "error" backend.log

# 查看最近 50 行日志
tail -50 backend.log

# 实时监控日志
tail -f backend.log | grep -i "error"
```

### 使用 logx 记录日志

```go
import "github.com/zeromicro/go-zero/core/logx"

// 记录信息
logx.Infof("处理请求: %s", req.ID)

// 记录警告
logx.Errorf("查询失败: %v", err)

// 带上下文日志（推荐，便于追踪）
logx.WithContext(r.Context()).Errorf("请求处理失败: %v", err)

// 记录结构化数据
logx.WithContext(r.Context()).Infow("用户登录",
    logx.Field("userId", user.ID),
    logx.Field("ip", r.RemoteAddr),
)
```

---

## 能力二：前端调试

### 常见问题排查表

| 问题现象 | 排查方法 |
|----------|----------|
| 页面空白 | DevTools Console 查看 JS 错误 |
| 数据不显示 | Network 面板检查 API 响应 |
| 样式异常 | Elements 面板检查 CSS |
| 状态问题 | Vue DevTools 检查 Store 状态 |
| 路由问题 | Vue DevTools 检查 Router |

### API 响应处理（重要！）

**问题**: 数据显示为空或 `undefined`

**原因**: 前端 `request` 函数已提取 `response.data.data`

```typescript
// ✅ 正确：res 已经是 data 对象
const res = await request({ url: '/api/xxx' });
data.value = res.list;

// ❌ 错误：多了一层 .data
data.value = res.data.list;  // res.data 是 undefined!
```

### 类型错误排查

```typescript
// 常见错误：data.filter is not a function
// 原因：data 可能是 undefined 或非数组

// ✅ 添加空值检查
const list = res?.list || [];
data.value = list.filter(item => item.status === 1);

// ✅ 初始化默认值
const data = ref<MyType[]>([]);
```

### Vue DevTools 使用

1. 安装 Vue DevTools 浏览器扩展
2. 打开 DevTools → Vue 面板
3. 检查组件树、Props、State
4. 使用 Timeline 追踪事件

---

## 能力三：数据库调试

### 常见问题

| 错误 | 原因 | 解决方案 |
|------|------|----------|
| `column "deleted_at" does not exist` | 模型移除软删除但数据库有该列 | 执行迁移删除列 |
| `duplicate key violates unique constraint` | 唯一索引冲突 | 检查重复数据 |
| `foreign key constraint fails` | 外键约束冲突 | 检查关联数据或调整 ON DELETE 策略 |

### 查看表结构

```sql
-- PostgreSQL 查看表结构
\d+ table_name

-- 查看所有表
\dt

-- 查看索引
\di
```

### GORM 调试模式

```go
// 开启 SQL 日志
db.Debug().Where("id = ?", id).Find(&user)

// 全局开启
db = db.Debug()
```

---

## 能力四：通知系统调试

### 通知未发送排查

1. **检查渠道是否启用**
   ```sql
   SELECT name, enabled, events FROM notification_channels;
   ```

2. **检查事件匹配**
   - 渠道 events 配置是否包含目标事件
   - 通配符是否正确（`system.*` 匹配 `system.started`）

3. **检查发送日志**
   ```sql
   SELECT * FROM notification_logs
   ORDER BY created_at DESC LIMIT 10;
   ```

4. **检查错误信息**
   ```sql
   SELECT error, status FROM notification_logs
   WHERE status = 'failed';
   ```

### Telegram 调试

```bash
# 验证 Bot Token
curl https://api.telegram.org/bot<TOKEN>/getMe

# 获取 Chat ID
curl https://api.telegram.org/bot<TOKEN>/getUpdates

# 测试发送消息
curl -X POST "https://api.telegram.org/bot<TOKEN>/sendMessage" \
  -d "chat_id=<CHAT_ID>&text=测试消息"
```

---

## 能力五：API 请求追踪

### go-zero 请求日志

go-zero 默认记录所有请求日志，格式：
```
@timestamp method=GET path=/api/xxx duration=15ms response=200
```

### 添加自定义追踪

```go
// Handler 中添加
logx.WithContext(r.Context()).Infof("收到请求: %+v", req)

// Logic 中添加
logx.WithContext(l.ctx).Infof("处理逻辑: %s", step)
```

### 前端请求拦截器

查看 `frontend/src/service/request/index.ts`：
- 请求拦截：添加 Token、日志
- 响应拦截：提取 data、错误处理

---

## 快速诊断清单

1. **后端是否运行？**
   ```bash
   curl http://localhost:8080/health
   ```

2. **API 返回格式正确？**
   - 检查是否有 `{code, msg, data}` 结构

3. **Token 有效？**
   - 检查 JWT 是否过期（默认 7 天）
   - 检查请求 Header 是否带 `Authorization: Bearer xxx`

4. **数据库连接正常？**
   - 查看后端启动日志
   - 检查 `config.yaml` 数据库配置

5. **前端代理配置？**
   - 检查 `vite.config.ts` 中的 proxy 设置
   - 开发环境应代理到 `http://localhost:8080`

---

## 导航速查

| 功能 | 路径 |
|------|------|
| **Handler** | `backend/internal/handler/` |
| **Logic** | `backend/internal/logic/` |
| **Model** | `backend/model/` |
| **配置文件** | `backend/etc/config.yaml` |
| **请求封装** | `frontend/src/service/request/` |
| **API 定义** | `frontend/src/service/api/` |
| **上下文文档** | `contexts/context.md` |
