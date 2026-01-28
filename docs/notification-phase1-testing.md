# 通知功能测试指南

## 阶段 1 完成状态

✅ **已完成的任务**:
1. ✅ 创建数据库表结构 (`001_create_notification_tables.sql`)
2. ✅ 创建数据模型 (NotificationChannel, NotificationRule, NotificationLog)
3. ✅ 定义核心接口 (Event, NotificationProvider, NotificationManager)
4. ✅ 实现 NotificationManager
5. ✅ 实现 Webhook 提供者
6. ✅ 扩展配置文件支持通知
7. ✅ 集成到 ServiceContext

## 测试步骤

### 1. 运行数据库迁移

```bash
# 进入 backend 目录
cd backend

# 连接到 PostgreSQL 数据库
psql -h 192.168.50.10 -U postgres -d logflux

# 执行 SQL 文件
\i scripts/migrations/001_create_notification_tables.sql

# 验证表是否创建成功
\dt notification_*

# 应该看到:
# notification_channels
# notification_rules
# notification_logs
```

### 2. 配置 Webhook 测试

使用 [webhook.site](https://webhook.site/) 获取测试 URL:

1. 访问 https://webhook.site/
2. 复制你的唯一 URL (例如: `https://webhook.site/12345678-1234-1234-1234-123456789abc`)
3. 更新 `backend/etc/config.yaml` 中的 Webhook URL:

```yaml
Notification:
  Enabled: true
  Channels:
    - Name: "webhook-default"
      Type: "webhook"
      Enabled: true
      Config:
        url: "https://webhook.site/your-unique-url"  # 替换为你的 URL
        method: "POST"
      Events:
        - "system.*"
        - "archive.failed"
```

### 3. 启动后端服务

```bash
cd backend
go run logflux.go -f etc/config.yaml
```

启动日志应该显示:
```
Registering notification providers...
Created notification channel: webhook-default
Created notification rule: 归档任务失败告警
Notification manager started successfully
```

### 4. 验证系统启动通知

启动后,应该会自动发送一条系统启动通知。

在 webhook.site 页面查看是否收到通知,内容类似:

```json
{
  "type": "system.startup",
  "level": "info",
  "title": "系统启动",
  "message": "LogFlux 系统已成功启动",
  "data": {},
  "timestamp": "2026-01-28T10:00:00Z"
}
```

### 5. 手动触发测试通知

可以修改 `backend/internal/tasks/archive.go` 临时添加测试代码:

```go
// 在 runArchive() 函数开始处添加
func (t *ArchiveTask) runArchive() {
    // 测试通知 - 模拟归档失败
    if t.notificationMgr != nil {
        event := notification.NewEvent(
            notification.EventArchiveFailed,
            notification.LevelError,
            "归档任务失败",
            "这是一个测试通知",
        ).WithData("error", "test error")

        t.notificationMgr.Notify(context.Background(), event)
    }

    // ... 原有代码
}
```

然后等待定时任务触发,或重启服务。

### 6. 查看通知历史

连接数据库查询通知历史:

```sql
-- 查看所有通知记录
SELECT id, channel_id, event_type, status, created_at, sent_at
FROM notification_logs
ORDER BY created_at DESC
LIMIT 10;

-- 查看失败的通知
SELECT *
FROM notification_logs
WHERE status = 'failed'
ORDER BY created_at DESC;

-- 统计通知发送情况
SELECT
    status,
    COUNT(*) as count
FROM notification_logs
GROUP BY status;
```

## 故障排查

### 问题 1: 数据库表未创建

**错误信息**: `relation "notification_channels" does not exist`

**解决方法**:
```bash
psql -h 192.168.50.10 -U postgres -d logflux \
  -f backend/scripts/migrations/001_create_notification_tables.sql
```

### 问题 2: Webhook 发送失败

**错误信息**: `failed to send request: ...`

**检查**:
1. URL 是否正确
2. 网络是否可达
3. webhook.site 是否正常

### 问题 3: 通知管理器未启动

**错误信息**: `notification manager not started`

**检查**:
1. `config.yaml` 中 `Notification.Enabled` 是否为 `true`
2. 启动日志是否有错误信息

## 下一步

阶段 1 (基础设施) 已完成! 下一步是阶段 2 (核心功能):

1. 实现 Email 提供者
2. 实现 Telegram 提供者
3. 实现基础规则引擎
4. 集成到关键事件点

参考文档:
- [完整设计文档](../notification-feature-design.md)
- [任务清单](../notification-task-checklist.md)
- [快速参考](../notification-quick-reference.md)

## 已创建的文件

### 数据库
- `backend/scripts/migrations/001_create_notification_tables.sql`

### 模型
- `backend/model/notification_channel.go`
- `backend/model/notification_rule.go`
- `backend/model/notification_log.go`

### 核心代码
- `backend/internal/notification/event.go`
- `backend/internal/notification/provider.go`
- `backend/internal/notification/notification.go`
- `backend/internal/notification/manager.go`

### 提供者
- `backend/internal/notification/providers/webhook.go`

### 配置和集成
- `backend/internal/config/config.go` (已更新)
- `backend/etc/config.yaml` (已更新)
- `backend/internal/svc/service_context.go` (已更新)

**总计**: 11 个文件 (4 个新增, 3 个更新, 4 个模型)
