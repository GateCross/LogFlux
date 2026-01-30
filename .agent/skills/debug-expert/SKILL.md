---
name: debug-expert
description: 调试专家技能，排查后端/前端问题。使用场景："API 返回错误"、"页面崩溃"、"日志分析"。
version: 1.0.0
---

# 调试专家

你是 LogFlux 的调试专家，擅长快速定位和解决问题。

---

## 能力一：后端 API 调试

### 常见错误及排查步骤

| 错误 | 可能原因 | 排查命令/位置 |
|------|---------|--------------|
| `backend request error` | Handler 未使用 `result.HttpResult` | 检查 `internal/handler/` 对应文件 |
| 401 Unauthorized | Token 过期或缺失 | 检查请求 Header 中的 `Authorization` |
| 500 Internal Error | Logic 层抛出异常 | 查看后端日志 `logx.Error` 输出 |
| 数据库错误 | Model 字段与数据库不匹配 | 检查 `backend/model/` 结构体 |

### 日志分析

```bash
# 启动后端（开发模式会输出详细日志）
cd backend && go run logflux.go -f etc/config.yaml

# 搜索错误日志
grep -i "error" backend.log
```

### 使用 logx 记录日志

```go
import "github.com/zeromicro/go-zero/core/logx"

// 记录信息
logx.Infof("处理请求: %s", req.ID)

// 记录错误
logx.Errorf("查询失败: %v", err)

// 记录带上下文
logx.WithContext(r.Context()).Errorf("请求处理失败")
```

---

## 能力二：前端调试

### 常见问题排查

| 问题 | 排查方法 |
|------|---------|
| 页面空白 | 打开 DevTools Console 查看 JS 错误 |
| 数据不显示 | Network 面板检查 API 响应 |
| 样式异常 | Elements 面板检查 CSS |
| 状态问题 | Vue DevTools 检查 Store 状态 |

### API 响应检查

```typescript
// 前端 request 函数已提取 response.data.data
const res = await request({ url: '/api/xxx' });

// ✅ 正确
data.value = res.list;

// ❌ 错误（多了一层 .data）
data.value = res.data.list;
```

### 开发者工具

```bash
# 启动前端开发服务器
cd frontend && pnpm dev

# 打开 http://localhost:3000
# 按 F12 打开 DevTools
```

---

## 能力三：API 请求追踪

### 后端请求日志

在 `logflux.go` 中，go-zero 默认记录请求日志。如需详细追踪：

```go
// 在 Handler 中添加
logx.WithContext(r.Context()).Infof("收到请求: %+v", req)
```

### 前端请求拦截

查看 `frontend/src/service/request/index.ts` 中的拦截器配置。

---

## 快速诊断清单

1. **后端是否运行？** → `curl http://localhost:8080/health`
2. **API 返回格式正确？** → 检查是否有 `{code, msg, data}` 结构
3. **Token 有效？** → 检查 JWT 是否过期
4. **数据库连接正常？** → 查看启动日志
5. **前端代理配置？** → 检查 `vite.config.ts` 中的 proxy 设置
