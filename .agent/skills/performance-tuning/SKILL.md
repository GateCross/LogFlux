---
name: performance-tuning
description: 性能优化技能。使用场景："API 响应慢"、"页面卡顿"、"内存占用高"。
version: 2.0.0
---

# 性能优化专家

你是 LogFlux 的性能优化专家，擅长识别和优化前后端性能瓶颈。

---

## 能力一：Go 后端性能分析

### pprof 性能剖析

```go
// 在 logflux.go 中添加（仅开发环境）
import _ "net/http/pprof"

// 性能分析端点自动可用
// http://localhost:8080/debug/pprof/
```

```bash
# CPU 分析（30 秒采样）
go tool pprof http://localhost:8080/debug/pprof/profile?seconds=30

# 内存分析
go tool pprof http://localhost:8080/debug/pprof/heap

# Goroutine 分析
go tool pprof http://localhost:8080/debug/pprof/goroutine

# 交互式 Web UI
go tool pprof -http=:6060 http://localhost:8080/debug/pprof/profile
```

### 常见优化点

| 问题 | 诊断方法 | 优化方案 |
|------|----------|----------|
| JSON 序列化慢 | pprof CPU | 使用 `sonic` 或 `json-iterator` |
| 对象分配频繁 | pprof heap | 使用 `sync.Pool` 复用对象 |
| 数据库查询慢 | EXPLAIN ANALYZE | 添加索引、批量查询 |
| Goroutine 泄露 | pprof goroutine | 确保 channel 正确关闭 |
| 中间件开销大 | trace | 使用 `sync.Pool` 复用 responseWriter |

### sync.Pool 最佳实践

```go
var bufferPool = sync.Pool{
    New: func() interface{} {
        return new(bytes.Buffer)
    },
}

func process() {
    buf := bufferPool.Get().(*bytes.Buffer)
    defer func() {
        buf.Reset()  // 重要: 使用前重置
        bufferPool.Put(buf)
    }()
    // 使用 buf...
}
```

### ResponseMiddleware 优化

项目中已实现响应包装中间件，确保使用 `sync.Pool`：

```go
// backend/internal/middleware/response_middleware.go
var rwPool = sync.Pool{
    New: func() interface{} {
        return &responseWriter{
            body: make([]byte, 0, 1024),  // 预分配
        }
    },
}
```

---

## 能力二：数据库优化

### GORM 查询优化

```go
// ❌ N+1 查询问题
for _, user := range users {
    db.Where("user_id = ?", user.ID).Find(&orders)
}

// ✅ 使用 Preload 预加载
db.Preload("Orders").Find(&users)

// ✅ 批量查询
db.Where("user_id IN ?", userIds).Find(&orders)

// ✅ 只选择需要的字段
db.Select("id", "name").Find(&users)
```

### PostgreSQL 慢查询分析

```sql
-- 启用查询统计
CREATE EXTENSION IF NOT EXISTS pg_stat_statements;

-- 查看慢查询 Top 10
SELECT query, calls, mean_time, total_time
FROM pg_stat_statements
ORDER BY mean_time DESC
LIMIT 10;

-- 执行计划分析
EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT)
SELECT * FROM notification_logs WHERE channel_id = 1;
```

### 索引优化

```sql
-- 检查索引使用情况
SELECT relname, indexrelname, idx_scan, idx_tup_read
FROM pg_stat_user_indexes
ORDER BY idx_scan DESC;

-- 查找缺失索引
SELECT schemaname, relname, seq_scan, seq_tup_read
FROM pg_stat_user_tables
WHERE seq_scan > 0
ORDER BY seq_tup_read DESC;
```

### 连接池配置

```go
// 在 ServiceContext 中配置
sqlDB, _ := db.DB()
sqlDB.SetMaxOpenConns(100)           // 最大连接数
sqlDB.SetMaxIdleConns(10)            // 最大空闲连接
sqlDB.SetConnMaxLifetime(time.Hour)  // 连接最大生命周期
sqlDB.SetConnMaxIdleTime(time.Minute * 10)  // 空闲连接超时
```

---

## 能力三：前端性能优化

### 性能检测工具

```bash
# Lighthouse 分析（Chrome DevTools → Lighthouse）
# 或命令行
npx lighthouse http://localhost:3000 --view

# Bundle 分析
cd frontend && pnpm build --mode analyze
```

### SoybeanAdmin 优化清单

| 问题 | 优化方案 |
|------|----------|
| 首屏加载慢 | 路由懒加载（已默认启用） |
| 列表卡顿 | `naive-ui` 的 `VirtualList` 虚拟滚动 |
| 图片加载慢 | 懒加载 + WebP 格式 |
| 重复渲染 | 使用 `computed`、`shallowRef` |
| 状态管理慢 | 使用 `pinia` 的 `storeToRefs` |

### 表格虚拟滚动

```vue
<template>
  <n-data-table
    :data="data"
    :columns="columns"
    :max-height="400"
    :virtual-scroll="true"  <!-- 启用虚拟滚动 -->
  />
</template>
```

### 响应式数据优化

```typescript
// ❌ 大对象使用 ref（深层响应式开销大）
const bigData = ref(largeArray)

// ✅ 使用 shallowRef（只追踪 .value 变化）
const bigData = shallowRef(largeArray)

// ✅ 频繁更新的状态使用 shallowReactive
const state = shallowReactive({ list: [], loading: false })
```

---

## 能力四：API 响应优化

### 分页查询优化

```go
// ✅ 使用 LIMIT/OFFSET 分页
db.Offset((page - 1) * pageSize).Limit(pageSize).Find(&items)

// ✅ 大数据量使用游标分页
db.Where("id > ?", lastId).Limit(pageSize).Find(&items)
```

### 响应数据精简

```go
// ❌ 返回完整模型
db.Find(&users)

// ✅ 只返回必要字段
db.Select("id", "username", "status").Find(&users)

// ✅ 使用 DTO 结构
type UserListItem struct {
    ID       uint   `json:"id"`
    Username string `json:"username"`
}
```

### 并发请求处理

```go
// 并行获取多个数据源
var (
    users  []User
    orders []Order
    wg     sync.WaitGroup
)

wg.Add(2)
go func() {
    defer wg.Done()
    db.Find(&users)
}()
go func() {
    defer wg.Done()
    db.Find(&orders)
}()
wg.Wait()
```

---

## 能力五：性能基准测试

### Go 基准测试

```bash
# 运行所有基准测试
cd backend && go test -bench=. -benchmem ./...

# 对比两次测试结果
go test -bench=. -count=5 > old.txt
# 修改代码后
go test -bench=. -count=5 > new.txt
benchstat old.txt new.txt
```

### 编写基准测试

```go
func BenchmarkEventMatching(b *testing.B) {
    manager := NewManager(db, nil)
    event := &Event{Type: "system.started"}
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        manager.matchChannels(event)
    }
}
```

---

## 监控指标导航

| 指标 | 检查位置 |
|------|----------|
| API 响应时间 | go-zero 内置日志 |
| 数据库慢查询 | pg_stat_statements |
| 内存使用 | pprof heap |
| Goroutine 数量 | pprof goroutine |
| 前端 FCP/LCP | Lighthouse |
| Bundle 大小 | Vite build 输出 |

---

## 快速诊断清单

1. **API 慢？** → 检查数据库查询（EXPLAIN ANALYZE）
2. **内存高？** → 运行 pprof heap 分析
3. **CPU 高？** → 运行 pprof profile 分析
4. **前端卡？** → 打开 DevTools Performance 面板
5. **首屏慢？** → 检查路由懒加载和代码分割
