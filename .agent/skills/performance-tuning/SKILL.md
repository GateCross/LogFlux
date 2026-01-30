---
name: performance-tuning
description: 性能优化技能。使用场景："API 响应慢"、"页面卡顿"、"内存占用高"。
version: 1.0.0
---

# 性能优化专家

你是 LogFlux 的性能优化专家，擅长识别和优化性能瓶颈。

---

## 能力一：Go 后端性能分析

### pprof 分析

```go
// 在 logflux.go 中添加（开发环境）
import _ "net/http/pprof"

// 访问性能分析端点
// http://localhost:8080/debug/pprof/
```

```bash
# CPU 分析
go tool pprof http://localhost:8080/debug/pprof/profile?seconds=30

# 内存分析
go tool pprof http://localhost:8080/debug/pprof/heap

# Goroutine 分析
go tool pprof http://localhost:8080/debug/pprof/goroutine
```

### 常见优化点

| 问题 | 优化方案 |
|------|---------|
| JSON 序列化慢 | 使用 `sonic` 或 `json-iterator` |
| 对象分配频繁 | 使用 `sync.Pool` 复用对象 |
| 数据库查询慢 | 添加索引、使用批量查询 |
| Goroutine 泄露 | 确保 channel 正确关闭 |

### sync.Pool 使用示例

```go
var bufferPool = sync.Pool{
    New: func() interface{} {
        return new(bytes.Buffer)
    },
}

func process() {
    buf := bufferPool.Get().(*bytes.Buffer)
    defer func() {
        buf.Reset()
        bufferPool.Put(buf)
    }()
    // 使用 buf...
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
```

### 索引检查

```sql
-- PostgreSQL 查看慢查询
SELECT query, calls, mean_time
FROM pg_stat_statements
ORDER BY mean_time DESC
LIMIT 10;

-- 检查索引使用
EXPLAIN ANALYZE SELECT * FROM users WHERE username = 'admin';
```

### 连接池配置

```go
// 在 ServiceContext 中配置
sqlDB, _ := db.DB()
sqlDB.SetMaxOpenConns(100)      // 最大连接数
sqlDB.SetMaxIdleConns(10)       // 最大空闲连接
sqlDB.SetConnMaxLifetime(time.Hour) // 连接最大生命周期
```

---

## 能力三：前端性能优化

### 性能检测

```bash
# Lighthouse 分析
# 在 Chrome DevTools → Lighthouse 面板运行

# 或使用命令行
npx lighthouse http://localhost:3000 --view
```

### 常见优化

| 问题 | 优化方案 |
|------|---------|
| 首屏加载慢 | 路由懒加载、代码分割 |
| 列表卡顿 | 虚拟滚动（如 `naive-ui` 的 `VirtualList`） |
| 图片加载慢 | 懒加载、WebP 格式 |
| 重复渲染 | 使用 `computed`、`shallowRef` |

### Vite 打包分析

```bash
# 添加到 package.json scripts
"analyze": "vite build --mode analyze"

# 或使用 rollup-plugin-visualizer
```

---

## 能力四：中间件优化模式

### 响应缓冲优化

```go
// 使用 sync.Pool 复用 responseWriter
var rwPool = sync.Pool{
    New: func() interface{} {
        return &responseWriter{
            body: make([]byte, 0, 1024),
        }
    },
}
```

### 减少内存分配

```go
// ❌ 每次分配新切片
body := make([]byte, 0)

// ✅ 预分配合理大小
body := make([]byte, 0, 1024)

// ✅ 使用 bytes.Buffer
var buf bytes.Buffer
buf.Grow(1024)
```

---

## 性能基准测试

```bash
# 运行基准测试
cd backend && go test -bench=. -benchmem ./...

# 对比两次结果
go test -bench=. -count=5 > old.txt
# 修改代码后
go test -bench=. -count=5 > new.txt
benchstat old.txt new.txt
```
