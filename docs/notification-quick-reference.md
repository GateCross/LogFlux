# LogFlux 通知功能 - 快速参考

## 文档索引

1. **[notification-feature-design.md](./notification-feature-design.md)** - 完整设计文档
   - 功能需求
   - 架构设计
   - 数据模型
   - API 设计
   - 配置示例
   - 安全考虑

2. **[notification-task-checklist.md](./notification-task-checklist.md)** - 实施任务清单
   - 60 个具体任务
   - 6 个阶段规划
   - 时间估算
   - 优先级标注
   - 依赖关系

## 快速开始

### 1. 阶段规划 (9 周)

| 阶段 | 时间 | 目标 | 产出 |
|------|------|------|------|
| 阶段 1 | 第 1-2 周 | 基础设施 | Webhook 通知可用 |
| 阶段 2 | 第 3-4 周 | 核心功能 | Email、Telegram、基础规则引擎 |
| 阶段 3 | 第 5 周 | 管理 API | 完整的通知管理 API |
| 阶段 4 | 第 6-7 周 | 高级功能 | Slack、企业微信、高级规则 |
| 阶段 5 | 第 8 周 | 前端界面 | 管理界面 |
| 阶段 6 | 第 9 周 | 测试优化 | 测试、文档、上线 |

### 2. 核心组件

```
backend/
├── internal/notification/           # 通知核心模块
│   ├── manager.go                  # 通知管理器
│   ├── event.go                    # 事件定义
│   ├── providers/                  # 通知提供者
│   │   ├── webhook.go              # Webhook (阶段1)
│   │   ├── email.go                # Email (阶段2)
│   │   ├── telegram.go             # Telegram (阶段2)
│   │   ├── slack.go                # Slack (阶段4)
│   │   ├── wecom.go                # 企业微信 (阶段4)
│   │   └── dingtalk.go             # 钉钉 (阶段4)
│   ├── rules/                      # 规则引擎
│   │   ├── engine.go               # 规则引擎 (阶段2)
│   │   └── evaluator.go            # 评估器 (阶段2)
│   └── templates/                  # 模板系统
│       └── template.go             # 模板引擎 (阶段2)
├── model/
│   ├── notification_channel.go     # 通知渠道模型
│   ├── notification_rule.go        # 告警规则模型
│   └── notification_log.go         # 通知历史模型
└── internal/logic/notification/    # 通知 API Logic (阶段3)
```

### 3. 优先级任务 (P0 必须完成)

#### 阶段 1 (Week 1-2)
- [ ] 数据库表结构 (3 张表)
- [ ] 数据模型 (3 个 model)
- [ ] 核心接口定义
- [ ] NotificationManager 实现
- [ ] Webhook 提供者
- [ ] 配置文件集成
- [ ] ServiceContext 集成

#### 阶段 2 (Week 3-4)
- [ ] Email 提供者
- [ ] 基础规则引擎 (阈值、频率)
- [ ] 事件集成 (归档、系统、Caddy)
- [ ] 通知模板系统

#### 阶段 3 (Week 5)
- [ ] 通知渠道 CRUD API
- [ ] 告警规则 CRUD API
- [ ] API 文档

### 4. 关键文件

| 文件 | 作用 | 阶段 |
|------|------|------|
| `backend/scripts/migrations/001_create_notification_tables.sql` | 数据库表结构 | 1 |
| `backend/internal/notification/manager.go` | 通知管理器核心 | 1 |
| `backend/internal/notification/providers/webhook.go` | Webhook 提供者 | 1 |
| `backend/internal/notification/providers/email.go` | Email 提供者 | 2 |
| `backend/internal/notification/rules/engine.go` | 规则引擎 | 2 |
| `backend/internal/config/config.go` | 配置结构扩展 | 1 |
| `backend/api/notification.api` | API 定义 | 3 |

### 5. 事件触发点

| 文件 | 事件 | 优先级 |
|------|------|--------|
| `backend/internal/tasks/archive.go` | archive.failed, archive.completed | P0 |
| `backend/internal/svc/service_context.go` | system.startup, redis.connection_failed | P0 |
| `backend/internal/logic/caddy/update_caddy_config_logic.go` | caddy.config_update_failed | P0 |
| `backend/internal/ingest/caddy.go` | log.high_error_rate, log.parse_error | P1 |
| `backend/internal/logic/auth/login_logic.go` | security.brute_force | P1 |

### 6. 配置示例

```yaml
# backend/etc/config.yaml

Notification:
  Enabled: true

  # 默认渠道
  DefaultChannels:
    - "webhook-default"

  # 通知渠道配置
  Channels:
    - Name: "webhook-default"
      Type: "webhook"
      Enabled: true
      Config:
        Url: "https://your-webhook-url.com/notify"
        Method: "POST"
      Events:
        - "system.*"
        - "archive.failed"

    - Name: "email-admin"
      Type: "email"
      Enabled: true
      Config:
        SmtpHost: "smtp.gmail.com"
        SmtpPort: 587
        Username: "your-email@gmail.com"
        Password: "your-app-password"
        From: "LogFlux <your-email@gmail.com>"
        To: ["admin@example.com"]
      Events:
        - "system.error"
        - "archive.failed"

  # 告警规则
  Rules:
    - Name: "Archive Failed Alert"
      Enabled: true
      RuleType: "threshold"
      EventType: "archive.failed"
      Condition:
        Field: "error_count"
        Operator: ">="
        Value: 1
      ChannelIds: [1, 2]
```

### 7. API 端点

```
# 通知渠道
GET    /api/notification/channels          # 获取列表
POST   /api/notification/channels          # 创建
PUT    /api/notification/channels/:id      # 更新
DELETE /api/notification/channels/:id      # 删除
POST   /api/notification/channels/:id/test # 测试

# 告警规则
GET    /api/notification/rules              # 获取列表
POST   /api/notification/rules              # 创建
PUT    /api/notification/rules/:id          # 更新
DELETE /api/notification/rules/:id          # 删除
POST   /api/notification/rules/:id/toggle   # 启用/禁用

# 通知历史
GET    /api/notification/logs               # 获取历史
POST   /api/notification/logs/:id/retry     # 重试
```

### 8. 开发流程

#### 第一步: 搭建基础 (Day 1-3)
```bash
# 1. 创建数据库表
cd backend/scripts/migrations
# 编写 001_create_notification_tables.sql

# 2. 创建数据模型
cd backend/model
# 创建 notification_channel.go, notification_rule.go, notification_log.go

# 3. 创建核心接口
cd backend/internal/notification
# 创建 event.go, provider.go, notification.go
```

#### 第二步: 实现管理器 (Day 4-7)
```bash
# 1. 实现 NotificationManager
cd backend/internal/notification
# 创建 manager.go

# 2. 实现 Webhook Provider
cd backend/internal/notification/providers
# 创建 webhook.go

# 3. 集成到 ServiceContext
cd backend/internal/svc
# 修改 service_context.go
```

#### 第三步: 配置和测试 (Day 8-10)
```bash
# 1. 扩展配置
cd backend/internal/config
# 修改 config.go

# 2. 更新配置文件
cd backend/etc
# 修改 config.yaml

# 3. 测试
# 编写单元测试
# 手动测试 Webhook 通知
```

### 9. 里程碑检查点

**M1 - Week 2 完成标准:**
- [ ] 数据库表创建成功
- [ ] NotificationManager 可以初始化
- [ ] Webhook 通知发送成功
- [ ] 配置文件可以加载通知配置
- [ ] 归档失败事件可以触发通知

**M2 - Week 4 完成标准:**
- [ ] Email 通知发送成功
- [ ] Telegram 通知发送成功
- [ ] 阈值规则可以正确评估
- [ ] 频率规则可以正确评估
- [ ] 所有关键事件已集成

**M3 - Week 5 完成标准:**
- [ ] 通知渠道 CRUD API 可用
- [ ] 告警规则 CRUD API 可用
- [ ] API 文档已编写
- [ ] Postman 测试通过

### 10. 依赖包

```bash
# 添加依赖
cd backend
go get gopkg.in/gomail.v2                                          # Email
go get github.com/go-telegram-bot-api/telegram-bot-api/v5          # Telegram
go get github.com/antonmedv/expr                                   # 规则引擎
```

### 11. 测试清单

- [ ] Webhook 发送成功
- [ ] Webhook 发送失败重试
- [ ] Email 发送成功 (Gmail)
- [ ] Email 发送成功 (其他 SMTP)
- [ ] Telegram 发送成功
- [ ] 阈值规则触发
- [ ] 频率规则触发
- [ ] 规则静默时间生效
- [ ] 通知历史记录正确
- [ ] API CRUD 操作正确

### 12. 常见问题

**Q: 如何避免通知风暴?**
A: 使用 silence_duration (静默时间),同类型告警在静默时间内不重复发送。

**Q: 如何测试 Email 发送?**
A: 使用 Gmail 或 Mailtrap.io 测试账号。

**Q: 规则引擎如何工作?**
A: 使用 expr 库评估条件表达式,支持灵活的条件定义。

**Q: 通知失败如何处理?**
A: 记录到 notification_logs 表,支持手动重试。

### 13. 下一步行动

1. **立即开始**:
   - 创建数据库表结构
   - 定义核心接口

2. **本周完成**:
   - NotificationManager 基础实现
   - Webhook 提供者
   - 配置集成

3. **下周完成**:
   - Email 提供者
   - 基础规则引擎
   - 事件集成

---

## 联系方式

如有问题,请参考:
- [完整设计文档](./notification-feature-design.md)
- [任务清单](./notification-task-checklist.md)
