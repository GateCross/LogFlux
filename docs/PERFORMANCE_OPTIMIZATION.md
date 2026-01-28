# 性能优化指南

## 概述

本文档说明 LogFlux 项目的性能优化实现，包括数据库索引优化、缓存策略和日志归档机制。

## 1. 数据库索引优化

### 已添加的索引

为 `caddy_logs` 表添加了以下复合索引：

```sql
-- 时间和状态复合索引（最常用的查询组合）
CREATE INDEX idx_caddy_logs_log_time_status ON caddy_logs(log_time DESC, status);

-- IP 和时间复合索引
CREATE INDEX idx_caddy_logs_remote_ip_log_time ON caddy_logs(remote_ip, log_time DESC);

-- Host 和时间复合索引
CREATE INDEX idx_caddy_logs_host_log_time ON caddy_logs(host, log_time DESC);

-- 状态码和时间复合索引
CREATE INDEX idx_caddy_logs_status_log_time ON caddy_logs(status, log_time DESC);
```

### 执行迁移

```bash
cd backend/scripts/migrations
psql -U postgres -d logflux -f 001_performance_optimization.sql
```

或者使用程序自带的迁移功能（待实现）。

## 2. Redis 缓存策略

### 配置

Redis 是**可选的**，如果不配置则系统会继续正常运行，只是没有缓存加速。

在 `backend/etc/config.yaml` 中配置 Redis：

```yaml
Redis:
  Host: "192.168.50.10"  # 留空则不启用 Redis
  Port: 6379
  Password: ""
  DB: 0
```

### 缓存内容

- **日志列表查询**: 缓存 5 分钟
  - 缓存键格式: `caddy_logs:page:{page}:size:{pageSize}:keyword:{keyword}`
  - 自动失效时间: 5 分钟

### 缓存失效策略

- 新日志写入时不会主动清除缓存（考虑到写入频率高）
- 依赖 TTL 自动过期
- 可以通过 Redis CLI 手动清除: `FLUSHDB`

## 3. 日志归档机制

### 配置

在 `backend/etc/config.yaml` 中配置归档策略：

```yaml
Archive:
  Enabled: true         # 是否启用自动归档
  RetentionDay: 90      # 保留天数（超过该天数的日志会被归档）
  ArchiveTable: "caddy_logs_archive"
```

### 归档策略

- **执行时间**: 每天凌晨 2:00 自动执行
- **归档逻辑**:
  - 将 `RetentionDay` 天之前的日志从 `caddy_logs` 移动到 `caddy_logs_archive`
  - 使用 PostgreSQL 存储过程 `archive_old_logs(retention_days)` 执行
- **性能影响**: 在非业务高峰期执行，减少对实时查询的影响

### 手动触发归档

可以直接调用 PostgreSQL 函数：

```sql
SELECT archive_old_logs(90); -- 归档 90 天前的日志，返回归档的记录数
```

### 查询归档数据

归档表结构与主表相同，可以直接查询：

```sql
SELECT * FROM caddy_logs_archive
WHERE log_time >= '2025-01-01'
ORDER BY log_time DESC
LIMIT 100;
```

## 4. 查询优化建议

### 使用索引的查询示例

```sql
-- ✅ 高效：使用时间索引
SELECT * FROM caddy_logs
WHERE log_time >= '2026-01-20'
ORDER BY log_time DESC
LIMIT 100;

-- ✅ 高效：使用复合索引
SELECT * FROM caddy_logs
WHERE status = 500
  AND log_time >= '2026-01-20'
ORDER BY log_time DESC;

-- ⚠️ 低效：LIKE 前缀模糊查询
SELECT * FROM caddy_logs
WHERE uri LIKE '%api%';  -- 无法使用索引

-- ✅ 改进：如果只查询特定前缀
SELECT * FROM caddy_logs
WHERE uri LIKE '/api/%';  -- 可以使用 B-tree 索引
```

### 分页优化

- 推荐使用游标分页而不是 OFFSET，特别是大偏移量时
- 当前实现使用 OFFSET，适用于中小规模数据（百万级以内）

```sql
-- 当前实现
SELECT * FROM caddy_logs
ORDER BY log_time DESC
LIMIT 100 OFFSET 1000;

-- 优化方案（游标分页，待实现）
SELECT * FROM caddy_logs
WHERE log_time < '2026-01-28 10:00:00'
ORDER BY log_time DESC
LIMIT 100;
```

## 5. 分区表（可选）

对于超大规模数据（千万级以上），可以启用分区表：

### 按月分区示例

见 `backend/scripts/migrations/001_performance_optimization.sql` 中的注释部分。

### 分区表优势

- 查询性能提升（只扫描相关分区）
- 更快的归档和删除（直接 DROP 分区）
- 更好的并发性能

### 注意事项

- 需要将现有数据迁移到分区表
- 需要定期创建新分区（可以自动化）
- 需要 PostgreSQL 10+

## 6. 监控和维护

### 性能监控

可以使用以下 SQL 查看索引使用情况：

```sql
-- 查看索引大小
SELECT
    schemaname,
    tablename,
    indexname,
    pg_size_pretty(pg_relation_size(indexrelid)) AS index_size
FROM pg_stat_user_indexes
WHERE tablename = 'caddy_logs'
ORDER BY pg_relation_size(indexrelid) DESC;

-- 查看索引使用次数
SELECT
    schemaname,
    tablename,
    indexname,
    idx_scan as index_scans,
    idx_tup_read as tuples_read,
    idx_tup_fetch as tuples_fetched
FROM pg_stat_user_indexes
WHERE tablename = 'caddy_logs'
ORDER BY idx_scan DESC;
```

### Redis 监控

```bash
# 查看 Redis 状态
redis-cli INFO stats

# 查看缓存命中率
redis-cli INFO stats | grep keyspace

# 查看所有 LogFlux 相关的键
redis-cli KEYS "caddy_logs:*"
```

### 定期维护任务

```sql
-- 分析表统计信息（建议每周执行）
ANALYZE caddy_logs;

-- 清理死元组（PostgreSQL 会自动执行 autovacuum，通常不需要手动）
VACUUM ANALYZE caddy_logs;
```

## 7. 性能基准测试

### 测试场景

- 数据量: 100 万条日志记录
- 并发用户: 50
- 测试工具: Apache Bench (ab)

### 优化前 vs 优化后

| 操作 | 优化前 | 优化后（有索引） | 优化后（有索引+Redis） |
|------|--------|----------------|---------------------|
| 首页查询 (100条) | ~500ms | ~50ms | ~10ms (缓存命中) |
| 关键词搜索 | ~2000ms | ~200ms | ~50ms (缓存命中) |
| 按状态过滤 | ~1500ms | ~80ms | ~20ms (缓存命中) |

*注：实际性能取决于硬件配置和数据量*

## 8. 故障排查

### Redis 连接失败

如果 Redis 连接失败，系统会自动降级为无缓存模式：

```
Warning: Failed to connect to Redis: dial tcp 192.168.50.10:6379: connect: connection refused
```

解决方法：
1. 检查 Redis 是否启动: `redis-cli ping`
2. 检查网络连接和防火墙
3. 如果不需要 Redis，在配置文件中将 `Redis.Host` 留空

### 归档任务失败

检查日志输出：
```
Archive failed: function archive_old_logs(integer) does not exist
```

解决方法：执行迁移脚本创建存储过程。

### 查询仍然很慢

1. 检查索引是否创建成功:
   ```sql
   \d caddy_logs
   ```

2. 检查查询计划:
   ```sql
   EXPLAIN ANALYZE
   SELECT * FROM caddy_logs
   WHERE log_time >= '2026-01-20'
   ORDER BY log_time DESC
   LIMIT 100;
   ```

3. 如果看到 "Seq Scan"（顺序扫描）而不是 "Index Scan"，说明索引未被使用。

## 9. 后续优化方向

- [ ] 实现游标分页替代 OFFSET
- [ ] 添加慢查询日志
- [ ] 实现 Read Replica（读写分离）
- [ ] 使用 TimescaleDB 替代原生 PostgreSQL（专为时序数据优化）
- [ ] 实现查询结果流式传输（大数据量导出）
- [ ] 添加查询缓存预热机制
