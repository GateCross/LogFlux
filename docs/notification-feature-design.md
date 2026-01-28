# LogFlux 通知功能设计文档

## 1. 概述

为 LogFlux 添加通知功能,支持多种通知渠道(Webhook、Email、Telegram、Slack 等),实现日志异常、系统告警等事件的实时通知。

## 2. 功能需求

### 2.1 通知渠道

- **Webhook**: 通用 HTTP 回调,支持自定义 URL
- **Email**: SMTP 邮件通知
- **Telegram**: Telegram Bot 通知
- **Slack**: Slack Webhook 通知
- **企业微信**: 企业微信机器人 (可选)
- **钉钉**: 钉钉机器人 (可选)

### 2.2 通知事件类型

#### 系统事件
- `system.startup`: 系统启动
- `system.shutdown`: 系统关闭
- `system.error`: 系统错误
- `redis.connection_failed`: Redis 连接失败
- `database.connection_failed`: 数据库连接失败

#### 日志采集事件
- `log.parse_error`: 日志解析错误
- `log.ingest_failed`: 日志写入失败
- `log.high_error_rate`: 高错误率告警 (4xx/5xx)
- `log.suspicious_ip`: 可疑 IP 访问
- `log.collection_stopped`: 日志采集中断

#### 归档事件
- `archive.failed`: 归档任务失败
- `archive.completed`: 归档任务完成
- `archive.slow`: 归档任务耗时过长
- `archive.anomaly`: 归档数据量异常

#### Caddy 配置事件
- `caddy.config_update_failed`: Caddy 配置更新失败
- `caddy.config_update_success`: Caddy 配置更新成功
- `caddy.log_source_discovered`: 新日志源发现

#### 安全事件
- `security.login_failed`: 登录失败
- `security.brute_force`: 暴力破解检测
- `security.admin_login`: 管理员登录
- `security.permission_denied`: 权限拒绝

### 2.3 告警规则

支持基于规则的告警触发:

- **阈值规则**: 数值超过/低于某个阈值
- **频率规则**: 在时间窗口内事件发生次数
- **比率规则**: 百分比超过阈值 (如错误率)
- **模式匹配**: 正则表达式匹配
- **复合条件**: 多个条件的逻辑组合

## 3. 架构设计

### 3.1 目录结构

```
backend/
├── internal/
│   ├── notification/
│   │   ├── notification.go         # 通知管理器接口
│   │   ├── manager.go              # 通知管理器实现
│   │   ├── event.go                # 事件定义
│   │   ├── providers/              # 通知提供者
│   │   │   ├── provider.go         # 提供者接口
│   │   │   ├── webhook.go          # Webhook 提供者
│   │   │   ├── email.go            # Email 提供者
│   │   │   ├── telegram.go         # Telegram 提供者
│   │   │   ├── slack.go            # Slack 提供者
│   │   │   ├── wecom.go            # 企业微信提供者
│   │   │   └── dingtalk.go         # 钉钉提供者
│   │   ├── rules/                  # 告警规则引擎
│   │   │   ├── engine.go           # 规则引擎
│   │   │   ├── rule.go             # 规则定义
│   │   │   ├── evaluator.go        # 条件评估器
│   │   │   └── aggregator.go       # 聚合器 (时间窗口)
│   │   └── templates/              # 通知模板
│   │       ├── template.go         # 模板引擎
│   │       └── default.go          # 默认模板
│   ├── logic/
│   │   └── notification/           # 通知相关 API
│   │       ├── send_notification_logic.go
│   │       ├── get_notification_channels_logic.go
│   │       ├── create_notification_channel_logic.go
│   │       ├── update_notification_channel_logic.go
│   │       ├── delete_notification_channel_logic.go
│   │       ├── get_notification_rules_logic.go
│   │       ├── create_notification_rule_logic.go
│   │       ├── update_notification_rule_logic.go
│   │       └── delete_notification_rule_logic.go
│   └── handler/
│       └── notification/           # 通知 HTTP 处理器
│           └── notification_handler.go
├── model/
│   ├── notification_channel.go     # 通知渠道模型
│   ├── notification_rule.go        # 告警规则模型
│   └── notification_log.go         # 通知历史记录模型
└── api/
    └── notification.api            # API 定义
```

### 3.2 数据模型

#### 通知渠道 (notification_channels)

```sql
CREATE TABLE notification_channels (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    name VARCHAR(100) NOT NULL,
    type VARCHAR(50) NOT NULL,  -- webhook, email, telegram, slack, wecom, dingtalk
    enabled BOOLEAN NOT NULL DEFAULT TRUE,

    -- 配置 (JSON)
    config JSONB NOT NULL,
    -- Webhook: {"url": "...", "method": "POST", "headers": {...}}
    -- Email: {"smtp_host": "...", "smtp_port": 587, "username": "...", "password": "...", "from": "...", "to": [...]}
    -- Telegram: {"bot_token": "...", "chat_id": "..."}
    -- Slack: {"webhook_url": "..."}

    -- 事件过滤 (订阅的事件类型)
    events TEXT[] NOT NULL,

    -- 描述
    description TEXT,

    INDEX idx_type (type),
    INDEX idx_enabled (enabled)
);
```

#### 告警规则 (notification_rules)

```sql
CREATE TABLE notification_rules (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    name VARCHAR(100) NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,

    -- 规则类型: threshold, frequency, ratio, pattern, composite
    rule_type VARCHAR(50) NOT NULL,

    -- 条件表达式 (JSON)
    condition JSONB NOT NULL,
    -- Threshold: {"field": "status", "operator": ">=", "value": 500}
    -- Frequency: {"event": "log.parse_error", "count": 100, "window": "5m"}
    -- Ratio: {"numerator": "status >= 500", "denominator": "total", "threshold": 0.1}
    -- Pattern: {"field": "uri", "pattern": "^/admin/.*"}

    -- 触发事件类型
    event_type VARCHAR(100) NOT NULL,

    -- 通知渠道 (关联 notification_channels)
    channel_ids BIGINT[] NOT NULL,

    -- 通知模板 (可选)
    template TEXT,

    -- 静默时间 (秒) - 避免告警风暴
    silence_duration INTEGER DEFAULT 300,

    -- 最后触发时间
    last_triggered_at TIMESTAMP,

    -- 触发次数
    trigger_count INTEGER DEFAULT 0,

    -- 描述
    description TEXT,

    INDEX idx_event_type (event_type),
    INDEX idx_enabled (enabled)
);
```

#### 通知历史 (notification_logs)

```sql
CREATE TABLE notification_logs (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),

    channel_id BIGINT NOT NULL REFERENCES notification_channels(id),
    rule_id BIGINT REFERENCES notification_rules(id),

    event_type VARCHAR(100) NOT NULL,
    event_data JSONB,

    status VARCHAR(50) NOT NULL,  -- pending, success, failed
    error_message TEXT,

    sent_at TIMESTAMP,

    INDEX idx_channel_id (channel_id),
    INDEX idx_rule_id (rule_id),
    INDEX idx_event_type (event_type),
    INDEX idx_status (status),
    INDEX idx_created_at (created_at)
);
```

### 3.3 核心接口

#### NotificationProvider 接口

```go
type NotificationProvider interface {
    // 发送通知
    Send(ctx context.Context, event *Event) error

    // 验证配置
    Validate(config map[string]interface{}) error

    // 获取提供者类型
    Type() string
}
```

#### NotificationManager 接口

```go
type NotificationManager interface {
    // 发送通知
    Notify(ctx context.Context, event *Event) error

    // 注册提供者
    RegisterProvider(provider NotificationProvider) error

    // 评估规则并触发通知
    EvaluateRules(ctx context.Context, data map[string]interface{}) error

    // 启动/停止
    Start(ctx context.Context) error
    Stop() error
}
```

#### Event 定义

```go
type Event struct {
    Type      string                 // 事件类型
    Level     string                 // 级别: info, warning, error, critical
    Title     string                 // 标题
    Message   string                 // 消息内容
    Data      map[string]interface{} // 事件数据
    Timestamp time.Time              // 时间戳
}
```

## 4. 配置示例

### 4.1 config.yaml

```yaml
Name: logflux-api
Host: 0.0.0.0
Port: 8888

# ... 其他配置

# 通知配置
Notification:
  Enabled: true

  # 默认通知渠道
  DefaultChannels:
    - "webhook-default"
    - "email-admin"

  # 通知提供者配置 (可选,也可以通过 API 管理)
  Channels:
    - Name: "webhook-default"
      Type: "webhook"
      Enabled: true
      Config:
        Url: "https://your-webhook-url.com/notify"
        Method: "POST"
        Headers:
          Content-Type: "application/json"
          Authorization: "Bearer your-token"
      Events:
        - "system.*"
        - "archive.failed"
        - "log.high_error_rate"

    - Name: "email-admin"
      Type: "email"
      Enabled: true
      Config:
        SmtpHost: "smtp.gmail.com"
        SmtpPort: 587
        Username: "your-email@gmail.com"
        Password: "your-app-password"
        From: "LogFlux <your-email@gmail.com>"
        To:
          - "admin@example.com"
          - "ops@example.com"
      Events:
        - "system.error"
        - "archive.failed"
        - "security.*"

    - Name: "telegram-alerts"
      Type: "telegram"
      Enabled: true
      Config:
        BotToken: "your-bot-token"
        ChatId: "your-chat-id"
      Events:
        - "log.high_error_rate"
        - "security.brute_force"

  # 告警规则 (可选,也可以通过 API 管理)
  Rules:
    - Name: "High 5xx Error Rate"
      Enabled: true
      RuleType: "ratio"
      EventType: "log.high_error_rate"
      Condition:
        Numerator: "status >= 500"
        Denominator: "total"
        Threshold: 0.05  # 5%
        Window: "5m"
      ChannelIds: [1, 2]
      SilenceDuration: 300  # 5分钟内不重复告警

    - Name: "Archive Task Failed"
      Enabled: true
      RuleType: "threshold"
      EventType: "archive.failed"
      Condition:
        Field: "error_count"
        Operator: ">="
        Value: 1
      ChannelIds: [2]
      Template: |
        归档任务失败

        时间: {{.Timestamp}}
        错误: {{.Data.error}}
        数据库: {{.Data.database}}

    - Name: "Brute Force Detection"
      Enabled: true
      RuleType: "frequency"
      EventType: "security.brute_force"
      Condition:
        Event: "security.login_failed"
        Count: 5
        Window: "1m"
        GroupBy: "remote_ip"
      ChannelIds: [1, 3]
      SilenceDuration: 600
```

## 5. API 设计

### 5.1 通知渠道管理

```
# 获取通知渠道列表
GET /api/notification/channels
Response: {
  "list": [...],
  "total": 10
}

# 创建通知渠道
POST /api/notification/channels
Request: {
  "name": "webhook-test",
  "type": "webhook",
  "enabled": true,
  "config": {...},
  "events": ["system.*"]
}

# 更新通知渠道
PUT /api/notification/channels/:id
Request: {...}

# 删除通知渠道
DELETE /api/notification/channels/:id

# 测试通知渠道
POST /api/notification/channels/:id/test
```

### 5.2 告警规则管理

```
# 获取告警规则列表
GET /api/notification/rules

# 创建告警规则
POST /api/notification/rules

# 更新告警规则
PUT /api/notification/rules/:id

# 删除告警规则
DELETE /api/notification/rules/:id

# 启用/禁用规则
POST /api/notification/rules/:id/toggle
```

### 5.3 通知历史

```
# 获取通知历史
GET /api/notification/logs
Query: ?channel_id=1&status=success&page=1&page_size=20

# 重新发送失败的通知
POST /api/notification/logs/:id/retry
```

## 6. 实施计划

### 阶段 1: 基础设施 (第 1-2 周)

**目标**: 建立通知系统基础框架

- [ ] 创建数据库表结构
- [ ] 实现 Event 和 Provider 接口
- [ ] 实现 NotificationManager
- [ ] 实现 Webhook 提供者
- [ ] 配置文件集成
- [ ] ServiceContext 集成

**产出**:
- `internal/notification/` 基础代码
- 数据库 migration 脚本
- Webhook 通知可用

### 阶段 2: 核心功能 (第 3-4 周)

**目标**: 实现主要通知渠道和基础规则引擎

- [ ] 实现 Email 提供者
- [ ] 实现 Telegram 提供者
- [ ] 实现基础规则引擎 (阈值、频率规则)
- [ ] 实现通知模板系统
- [ ] 集成到关键事件点 (归档、系统错误等)
- [ ] 通知历史记录

**产出**:
- Email、Telegram 通知可用
- 基础告警规则功能
- 系统关键事件通知

### 阶段 3: 管理 API (第 5 周)

**目标**: 提供通知配置管理 API

- [ ] 通知渠道 CRUD API
- [ ] 告警规则 CRUD API
- [ ] 通知历史查询 API
- [ ] 测试通知 API
- [ ] API 文档

**产出**:
- 完整的通知管理 API
- API 文档

### 阶段 4: 高级功能 (第 6-7 周)

**目标**: 实现高级特性

- [ ] Slack 提供者
- [ ] 企业微信/钉钉提供者
- [ ] 复杂规则引擎 (复合条件、模式匹配)
- [ ] 日志异常检测 (高错误率、可疑 IP)
- [ ] 通知静默时间 (避免告警风暴)
- [ ] 通知分组和批量发送

**产出**:
- 更多通知渠道
- 高级告警规则
- 日志异常检测

### 阶段 5: 前端界面 (第 8 周)

**目标**: 提供前端管理界面

- [ ] 通知渠道配置页面
- [ ] 告警规则配置页面
- [ ] 通知历史查看页面
- [ ] 实时通知展示 (WebSocket)

**产出**:
- 完整的前端管理界面

### 阶段 6: 测试与优化 (第 9 周)

**目标**: 测试、优化和文档

- [ ] 单元测试
- [ ] 集成测试
- [ ] 性能测试和优化
- [ ] 用户文档
- [ ] 部署文档

**产出**:
- 测试覆盖率 > 80%
- 完整文档

## 7. 技术选型

### 7.1 Go 依赖包

```go
// go.mod
require (
    // Email
    gopkg.in/gomail.v2 v2.0.0-20160411212932-81ebce5c23df

    // Telegram
    github.com/go-telegram-bot-api/telegram-bot-api/v5 v5.5.1

    // 表达式评估 (规则引擎)
    github.com/antonmedv/expr v1.15.5

    // 模板引擎 (已有)
    // text/template (标准库)
)
```

### 7.2 数据库

- PostgreSQL (已有)
- JSONB 存储配置和条件

### 7.3 缓存

- Redis (已有) - 用于规则状态缓存、静默时间控制

## 8. 事件集成点

### 8.1 日志采集 (internal/ingest/caddy.go)

```go
// 在日志解析失败时
if err := i.parseAndStore(line.Text); err != nil {
    i.notificationMgr.Notify(ctx, &notification.Event{
        Type:    "log.parse_error",
        Level:   "warning",
        Title:   "日志解析失败",
        Message: fmt.Sprintf("无法解析日志: %s", line.Text),
        Data:    map[string]interface{}{"error": err.Error()},
    })
}

// 检测高错误率
if errorRate > threshold {
    i.notificationMgr.Notify(ctx, &notification.Event{
        Type:    "log.high_error_rate",
        Level:   "error",
        Title:   "高错误率告警",
        Message: fmt.Sprintf("5xx 错误率: %.2f%%", errorRate*100),
        Data:    map[string]interface{}{"rate": errorRate},
    })
}
```

### 8.2 归档任务 (internal/tasks/archive.go)

```go
// 归档失败
if err != nil {
    notificationMgr.Notify(ctx, &notification.Event{
        Type:    "archive.failed",
        Level:   "error",
        Title:   "归档任务失败",
        Message: fmt.Sprintf("归档失败: %v", err),
        Data:    map[string]interface{}{"error": err.Error()},
    })
    return
}

// 归档完成
notificationMgr.Notify(ctx, &notification.Event{
    Type:    "archive.completed",
    Level:   "info",
    Title:   "归档任务完成",
    Message: fmt.Sprintf("已归档 %d 条记录", archivedCount),
    Data:    map[string]interface{}{
        "count":    archivedCount,
        "duration": duration.String(),
    },
})
```

### 8.3 Caddy 配置 (internal/logic/caddy/update_caddy_config_logic.go)

```go
// 配置更新失败
if httpResp.StatusCode != 200 {
    notificationMgr.Notify(ctx, &notification.Event{
        Type:    "caddy.config_update_failed",
        Level:   "error",
        Title:   "Caddy 配置更新失败",
        Message: fmt.Sprintf("状态码: %d", httpResp.StatusCode),
        Data:    map[string]interface{}{"status": httpResp.StatusCode},
    })
}
```

### 8.4 登录失败 (internal/logic/auth/login_logic.go)

```go
// 密码错误
if !comparePassword(user.Password, req.Password) {
    // 记录失败次数并检测暴力破解
    failCount := incrementLoginFailCount(req.Username, remoteIP)
    if failCount >= 5 {
        notificationMgr.Notify(ctx, &notification.Event{
            Type:    "security.brute_force",
            Level:   "critical",
            Title:   "暴力破解检测",
            Message: fmt.Sprintf("IP %s 尝试登录失败 %d 次", remoteIP, failCount),
            Data:    map[string]interface{}{
                "username": req.Username,
                "ip":       remoteIP,
                "count":    failCount,
            },
        })
    }
}
```

## 9. 通知模板示例

### 9.1 默认模板

```go
const DefaultTemplate = `
【{{.Level | upper}}】{{.Title}}

时间: {{.Timestamp | datetime}}
消息: {{.Message}}

{{- if .Data}}
详细信息:
{{- range $key, $value := .Data}}
  {{$key}}: {{$value}}
{{- end}}
{{- end}}

---
LogFlux 通知系统
`
```

### 9.2 Markdown 模板 (Telegram/Slack)

```markdown
**[{{.Level | upper}}] {{.Title}}**

⏰ 时间: `{{.Timestamp | datetime}}`
📝 消息: {{.Message}}

{{- if .Data}}
**详细信息:**
{{- range $key, $value := .Data}}
• *{{$key}}*: `{{$value}}`
{{- end}}
{{- end}}

---
_LogFlux 通知系统_
```

## 10. 安全考虑

1. **敏感信息加密**: SMTP 密码、Bot Token 等存储加密
2. **权限控制**: 通知配置需要管理员权限
3. **速率限制**: 防止通知风暴,限制发送频率
4. **日志脱敏**: 通知中不包含敏感数据 (如密码)
5. **HTTPS**: Webhook 使用 HTTPS
6. **Token 验证**: Webhook 支持签名验证

## 11. 监控指标

- 通知发送成功率
- 通知发送延迟
- 规则触发次数
- 通知渠道可用性
- 告警静默次数

## 12. 扩展性考虑

1. **插件化设计**: 新通知渠道可轻松添加
2. **规则引擎**: 支持自定义规则类型
3. **模板系统**: 支持自定义模板
4. **水平扩展**: 通知发送可异步化,支持队列
5. **多租户**: 预留租户隔离字段

## 13. 参考资料

- [Prometheus Alertmanager](https://prometheus.io/docs/alerting/latest/alertmanager/)
- [Grafana Alerting](https://grafana.com/docs/grafana/latest/alerting/)
- [PagerDuty Event API](https://developer.pagerduty.com/docs/events-api-v2/overview/)
