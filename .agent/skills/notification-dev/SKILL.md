---
name: notification-dev
description: 通知系统开发专家。使用场景："添加通知渠道"、"规则引擎配置"、"通知发送问题"、"Telegram/Email 配置"。
version: 1.0.0
---

# 通知系统开发专家

你是 LogFlux 通知系统的开发专家，深入理解事件驱动架构和多渠道通知机制。

---

## 项目概览

LogFlux 的通知系统支持多渠道告警推送，包括 Email、Telegram、Webhook 等。

### 核心组件

| 组件 | 路径 | 职责 |
|------|------|------|
| **NotificationManager** | `backend/internal/notification/manager.go` | 通知管理器核心 |
| **RuleEngine** | `backend/internal/notification/rule_engine.go` | 规则匹配引擎 |
| **Providers** | `backend/internal/notification/providers/` | 渠道实现 |
| **Models** | `backend/model/notification_*.go` | 数据模型 |

### 已实现的渠道

| 渠道 | 文件 | 配置项 |
|------|------|--------|
| **Webhook** | `providers/webhook.go` | `url`, `method`, `headers` |
| **Email** | `providers/email.go` | `host`, `port`, `username`, `password`, `from`, `to` |
| **Telegram** | `providers/telegram.go` | `bot_token`, `chat_id` |

---

## 能力一：添加新通知渠道

### 工作流

1. **创建 Provider 文件**
   ```bash
   创建 backend/internal/notification/providers/<channel_name>.go
   ```

2. **实现 NotificationProvider 接口**
   ```go
   type NotificationProvider interface {
       Type() string
       Send(ctx context.Context, event *Event, config map[string]interface{}) error
       Validate(config map[string]interface{}) error
   }
   ```

3. **注册到 Manager**
   编辑 `backend/internal/svc/service_context.go`：
   ```go
   // 在 initNotificationManager() 中注册
   notifMgr.RegisterProvider(providers.NewMyProvider())
   ```

4. **扩展配置结构**
   在 `backend/model/notification_channel.go` 中添加配置结构体：
   ```go
   type MyChannelConfig struct {
       ApiKey  string `json:"api_key"`
       Webhook string `json:"webhook"`
   }
   ```

5. **添加单元测试**
   创建 `providers/<channel_name>_test.go`

---

## 能力二：配置 Telegram 通知

### 快速配置步骤

1. **获取 Bot Token**
   - 在 Telegram 找 @BotFather
   - 发送 `/newbot` 创建机器人
   - 获取 Token: `123456789:ABCdefGHIjklMNOpqrsTUVwxyz`

2. **获取 Chat ID**
   - 向机器人发送消息
   - 访问 `https://api.telegram.org/bot<TOKEN>/getUpdates`
   - 在 JSON 中找 `chat.id`

3. **配置到系统**

   **方法 A：配置文件** (`backend/etc/config.yaml`)
   ```yaml
   notification:
     channels:
       - name: "telegram-alerts"
         type: "telegram"
         enabled: true
         events: ["system.*", "error.*"]
         config:
           bot_token: "YOUR_BOT_TOKEN"
           chat_id: "YOUR_CHAT_ID"
   ```

   **方法 B：数据库**
   ```sql
   INSERT INTO notification_channels (name, type, enabled, config, events)
   VALUES (
     'telegram-alerts', 'telegram', true,
     '{"bot_token": "xxx", "chat_id": "123"}',
     ARRAY['system.*']
   );
   ```

### 详细文档

📄 完整配置指南: `docs/telegram-setup-guide.md`

---

## 能力三：规则引擎配置

### 规则类型

| 类型 | 说明 | 配置示例 |
|------|------|----------|
| **threshold** | 阈值比较 | `value > 100` |
| **frequency** | 频率限制 | 5 分钟内超过 10 次 |
| **pattern** | 正则匹配 | `error.*timeout` |

### 阈值规则示例

```yaml
notification:
  rules:
    - name: "高 CPU 告警"
      type: "threshold"
      enabled: true
      events: ["metric.cpu"]
      condition:
        field: "value"
        operator: ">"
        threshold: 80
      actions:
        - channel_name: "telegram-alerts"
```

### 频率规则示例

```yaml
- name: "错误频率告警"
  type: "frequency"
  events: ["error.*"]
  condition:
    count: 10
    window: "5m"  # 5 分钟内超过 10 次
```

### 支持的操作符

- `>`, `<`, `>=`, `<=`, `==`, `!=`

---

## 能力四：调试通知发送问题

### 常见问题排查

| 问题 | 可能原因 | 排查方法 |
|------|----------|----------|
| 通知未发送 | 渠道未启用 | 检查 `enabled: true` |
| 事件不匹配 | events 配置错误 | 确认通配符正确 (`system.*`) |
| Telegram 失败 | Token/ChatID 错误 | 验证 getUpdates 响应 |
| Email 失败 | SMTP 认证失败 | 检查用户名密码和端口 |

### 日志检查

```bash
# 查看通知相关日志
grep -i "notification" backend.log
grep -i "send" backend.log | grep -i "error"
```

### 测试通知发送

重启后端后，系统会自动发送 `system.started` 事件：

```bash
cd backend && go run logflux.go -f etc/config.yaml
# 应收到 "LogFlux 通知系统已启动" 消息
```

---

## 能力五：前端通知管理

### 页面路径

| 功能 | 路径 |
|------|------|
| 渠道管理 | `frontend/src/views/notification/channel/` |
| 规则管理 | `frontend/src/views/notification/rule/` |
| 模板编辑 | `frontend/src/views/notification/template/` |
| 日志查看 | `frontend/src/views/notification/log/` |
| 通知铃铛 | `frontend/src/layouts/modules/global-header/components/header-notification.vue` |

### API 文件

- 后端定义: `backend/api/notification.api`
- 前端封装: `frontend/src/service/api/notification.ts`

---

## 导航速查

| 功能 | 路径 |
|------|------|
| **通知管理器** | `backend/internal/notification/manager.go` |
| **规则引擎** | `backend/internal/notification/rule_engine.go` |
| **事件定义** | `backend/internal/notification/event.go` |
| **渠道提供者** | `backend/internal/notification/providers/` |
| **数据模型** | `backend/model/notification_*.go` |
| **配置文件** | `backend/etc/config.yaml` |
| **设计文档** | `docs/notification-feature-design.md` |
| **进度报告** | `docs/notification-progress-report.md` |

---

## 事件类型常量

```go
// 系统事件
EventTypeSystemStarted  = "system.started"
EventTypeSystemStopped  = "system.stopped"
EventTypeSystemHealthy  = "system.healthy"

// 日志事件
EventTypeLogError       = "log.error"
EventTypeLogWarning     = "log.warning"
EventTypeLogCritical    = "log.critical"

// 用户事件
EventTypeUserLogin      = "user.login"
EventTypeUserLogout     = "user.logout"
```

### 事件匹配规则

- `system.*` - 匹配所有系统事件
- `log.error` - 精确匹配
- `*` - 匹配所有事件
