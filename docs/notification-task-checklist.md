# LogFlux 通知功能实施任务清单

## 任务概览

总计: 60 个任务
预计时间: 9 周
优先级: P0 (必须) > P1 (重要) > P2 (可选)

---

## 阶段 1: 基础设施 (第 1-2 周)

### 1.1 数据库设计与迁移 [P0]

- [ ] **Task 1.1.1**: 创建 notification_channels 表结构
  - 文件: `backend/scripts/migrations/001_create_notification_tables.sql`
  - 字段: id, name, type, enabled, config, events, description
  - 索引: idx_type, idx_enabled

- [ ] **Task 1.1.2**: 创建 notification_rules 表结构
  - 文件: 同上
  - 字段: id, name, enabled, rule_type, condition, event_type, channel_ids, template, silence_duration
  - 索引: idx_event_type, idx_enabled

- [ ] **Task 1.1.3**: 创建 notification_logs 表结构
  - 文件: 同上
  - 字段: id, channel_id, rule_id, event_type, event_data, status, error_message, sent_at
  - 索引: idx_channel_id, idx_rule_id, idx_event_type, idx_status, idx_created_at

- [ ] **Task 1.1.4**: 编写数据库 migration 脚本
  - 文件: `backend/scripts/migrations/migrate.go`
  - 支持 up/down migration

### 1.2 数据模型 [P0]

- [ ] **Task 1.2.1**: 创建 NotificationChannel 模型
  - 文件: `backend/model/notification_channel.go`
  - 包含 GORM 标签和验证

- [ ] **Task 1.2.2**: 创建 NotificationRule 模型
  - 文件: `backend/model/notification_rule.go`

- [ ] **Task 1.2.3**: 创建 NotificationLog 模型
  - 文件: `backend/model/notification_log.go`

### 1.3 核心接口定义 [P0]

- [ ] **Task 1.3.1**: 定义 Event 结构体
  - 文件: `backend/internal/notification/event.go`
  - 字段: Type, Level, Title, Message, Data, Timestamp

- [ ] **Task 1.3.2**: 定义 NotificationProvider 接口
  - 文件: `backend/internal/notification/provider.go`
  - 方法: Send(), Validate(), Type()

- [ ] **Task 1.3.3**: 定义 NotificationManager 接口
  - 文件: `backend/internal/notification/notification.go`
  - 方法: Notify(), RegisterProvider(), EvaluateRules(), Start(), Stop()

### 1.4 NotificationManager 实现 [P0]

- [ ] **Task 1.4.1**: 实现 Manager 基础结构
  - 文件: `backend/internal/notification/manager.go`
  - 包含 providers map, channels, rules, db, redis

- [ ] **Task 1.4.2**: 实现 RegisterProvider() 方法
  - 动态注册通知提供者

- [ ] **Task 1.4.3**: 实现 Notify() 方法
  - 接收事件并分发到对应渠道
  - 记录通知日志

- [ ] **Task 1.4.4**: 实现 loadChannels() 方法
  - 从数据库加载通知渠道配置

- [ ] **Task 1.4.5**: 实现 loadRules() 方法
  - 从数据库加载告警规则

- [ ] **Task 1.4.6**: 实现 Start() 和 Stop() 方法
  - 启动/停止通知管理器

### 1.5 Webhook 提供者 [P0]

- [ ] **Task 1.5.1**: 实现 WebhookProvider 结构体
  - 文件: `backend/internal/notification/providers/webhook.go`
  - 配置: URL, Method, Headers

- [ ] **Task 1.5.2**: 实现 Send() 方法
  - HTTP 请求发送通知
  - 支持自定义 Headers
  - 错误重试机制

- [ ] **Task 1.5.3**: 实现 Validate() 方法
  - 验证 URL 格式
  - 验证 Method

- [ ] **Task 1.5.4**: 单元测试
  - 文件: `backend/internal/notification/providers/webhook_test.go`

### 1.6 配置文件集成 [P0]

- [ ] **Task 1.6.1**: 扩展 Config 结构体
  - 文件: `backend/internal/config/config.go`
  - 添加 NotificationConf 字段

- [ ] **Task 1.6.2**: 定义 NotificationConf 结构
  - 字段: Enabled, DefaultChannels, Channels, Rules

- [ ] **Task 1.6.3**: 更新示例配置文件
  - 文件: `backend/etc/config.yaml`
  - 添加 Notification 配置示例

### 1.7 ServiceContext 集成 [P0]

- [ ] **Task 1.7.1**: 在 ServiceContext 中添加 NotificationMgr 字段
  - 文件: `backend/internal/svc/service_context.go`

- [ ] **Task 1.7.2**: 初始化 NotificationManager
  - 在 NewServiceContext() 中创建实例
  - 注册默认提供者
  - 加载配置

- [ ] **Task 1.7.3**: 启动 NotificationManager
  - 在服务启动时调用 Start()

---

## 阶段 2: 核心功能 (第 3-4 周)

### 2.1 Email 提供者 [P0]

- [ ] **Task 2.1.1**: 添加 gomail 依赖
  - 文件: `backend/go.mod`
  - `go get gopkg.in/gomail.v2`

- [ ] **Task 2.1.2**: 实现 EmailProvider 结构体
  - 文件: `backend/internal/notification/providers/email.go`
  - 配置: SmtpHost, SmtpPort, Username, Password, From, To

- [ ] **Task 2.1.3**: 实现 Send() 方法
  - 使用 gomail 发送邮件
  - 支持 HTML 格式

- [ ] **Task 2.1.4**: 实现 Validate() 方法
  - 验证 SMTP 配置
  - 验证邮箱格式

- [ ] **Task 2.1.5**: 单元测试

### 2.2 Telegram 提供者 [P1]

- [ ] **Task 2.2.1**: 添加 telegram-bot-api 依赖
  - `go get github.com/go-telegram-bot-api/telegram-bot-api/v5`

- [ ] **Task 2.2.2**: 实现 TelegramProvider 结构体
  - 文件: `backend/internal/notification/providers/telegram.go`
  - 配置: BotToken, ChatId

- [ ] **Task 2.2.3**: 实现 Send() 方法
  - 支持 Markdown 格式

- [ ] **Task 2.2.4**: 实现 Validate() 方法

- [ ] **Task 2.2.5**: 单元测试

### 2.3 规则引擎基础 [P0]

- [ ] **Task 2.3.1**: 添加 expr 依赖
  - `go get github.com/antonmedv/expr`

- [ ] **Task 2.3.2**: 创建 RuleEngine 结构体
  - 文件: `backend/internal/notification/rules/engine.go`

- [ ] **Task 2.3.3**: 实现阈值规则评估器
  - 文件: `backend/internal/notification/rules/evaluator.go`
  - 支持: >, <, >=, <=, ==, !=

- [ ] **Task 2.3.4**: 实现频率规则评估器
  - 使用 Redis 记录事件计数
  - 时间窗口滑动

- [ ] **Task 2.3.5**: 实现规则缓存
  - 使用 Redis 缓存规则状态
  - 实现静默时间

- [ ] **Task 2.3.6**: 集成到 NotificationManager
  - 在 EvaluateRules() 中调用规则引擎

### 2.4 通知模板系统 [P1]

- [ ] **Task 2.4.1**: 创建 Template 引擎
  - 文件: `backend/internal/notification/templates/template.go`
  - 使用 text/template

- [ ] **Task 2.4.2**: 定义默认模板
  - 文件: `backend/internal/notification/templates/default.go`
  - 文本格式和 Markdown 格式

- [ ] **Task 2.4.3**: 实现模板渲染
  - 支持自定义函数 (datetime, upper 等)

- [ ] **Task 2.4.4**: 集成到 Provider
  - 在 Send() 前渲染模板

### 2.5 事件集成 [P0]

- [ ] **Task 2.5.1**: 归档任务事件
  - 文件: `backend/internal/tasks/archive.go`
  - 事件: archive.failed, archive.completed

- [ ] **Task 2.5.2**: 系统启动事件
  - 文件: `backend/internal/svc/service_context.go`
  - 事件: system.startup

- [ ] **Task 2.5.3**: Redis 连接失败事件
  - 文件: `backend/internal/svc/service_context.go`
  - 事件: redis.connection_failed

- [ ] **Task 2.5.4**: Caddy 配置更新事件
  - 文件: `backend/internal/logic/caddy/update_caddy_config_logic.go`
  - 事件: caddy.config_update_failed, caddy.config_update_success

### 2.6 通知历史记录 [P1]

- [ ] **Task 2.6.1**: 实现日志记录方法
  - 文件: `backend/internal/notification/manager.go`
  - 方法: logNotification()

- [ ] **Task 2.6.2**: 异步写入数据库
  - 使用 goroutine 避免阻塞

- [ ] **Task 2.6.3**: 日志清理定时任务
  - 清理 30 天前的日志

---

## 阶段 3: 管理 API (第 5 周)

### 3.1 API 定义 [P0]

- [ ] **Task 3.1.1**: 定义通知渠道 API
  - 文件: `backend/api/notification.api`
  - 端点: GET/POST/PUT/DELETE /api/notification/channels

- [ ] **Task 3.1.2**: 定义告警规则 API
  - 端点: GET/POST/PUT/DELETE /api/notification/rules

- [ ] **Task 3.1.3**: 定义通知历史 API
  - 端点: GET /api/notification/logs

- [ ] **Task 3.1.4**: 生成代码
  - `goctl api go -api notification.api -dir .`

### 3.2 通知渠道 API [P0]

- [ ] **Task 3.2.1**: 实现获取渠道列表 Logic
  - 文件: `backend/internal/logic/notification/get_notification_channels_logic.go`

- [ ] **Task 3.2.2**: 实现创建渠道 Logic
  - 文件: `backend/internal/logic/notification/create_notification_channel_logic.go`
  - 验证配置
  - 测试连接

- [ ] **Task 3.2.3**: 实现更新渠道 Logic
  - 文件: `backend/internal/logic/notification/update_notification_channel_logic.go`

- [ ] **Task 3.2.4**: 实现删除渠道 Logic
  - 文件: `backend/internal/logic/notification/delete_notification_channel_logic.go`
  - 检查是否被规则使用

- [ ] **Task 3.2.5**: 实现测试渠道 Logic
  - 文件: `backend/internal/logic/notification/test_notification_channel_logic.go`
  - 发送测试消息

### 3.3 告警规则 API [P0]

- [ ] **Task 3.3.1**: 实现获取规则列表 Logic
  - 文件: `backend/internal/logic/notification/get_notification_rules_logic.go`

- [ ] **Task 3.3.2**: 实现创建规则 Logic
  - 文件: `backend/internal/logic/notification/create_notification_rule_logic.go`
  - 验证条件表达式
  - 验证渠道 ID

- [ ] **Task 3.3.3**: 实现更新规则 Logic
  - 文件: `backend/internal/logic/notification/update_notification_rule_logic.go`

- [ ] **Task 3.3.4**: 实现删除规则 Logic
  - 文件: `backend/internal/logic/notification/delete_notification_rule_logic.go`

- [ ] **Task 3.3.5**: 实现启用/禁用规则 Logic
  - 文件: `backend/internal/logic/notification/toggle_notification_rule_logic.go`

### 3.4 通知历史 API [P1]

- [ ] **Task 3.4.1**: 实现获取历史 Logic
  - 文件: `backend/internal/logic/notification/get_notification_logs_logic.go`
  - 支持筛选和分页

- [ ] **Task 3.4.2**: 实现重试通知 Logic
  - 文件: `backend/internal/logic/notification/retry_notification_logic.go`

### 3.5 API 文档 [P1]

- [ ] **Task 3.5.1**: 编写 API 使用文档
  - 文件: `docs/api/notification.md`
  - 包含请求/响应示例

---

## 阶段 4: 高级功能 (第 6-7 周)

### 4.1 Slack 提供者 [P1]

- [ ] **Task 4.1.1**: 实现 SlackProvider
  - 文件: `backend/internal/notification/providers/slack.go`
  - 使用 Webhook URL

- [ ] **Task 4.1.2**: 支持 Slack 消息格式
  - Block Kit 格式

- [ ] **Task 4.1.3**: 单元测试

### 4.2 企业微信提供者 [P2]

- [ ] **Task 4.2.1**: 实现 WeComProvider
  - 文件: `backend/internal/notification/providers/wecom.go`
  - 企业微信机器人 Webhook

- [ ] **Task 4.2.2**: 单元测试

### 4.3 钉钉提供者 [P2]

- [ ] **Task 4.3.1**: 实现 DingTalkProvider
  - 文件: `backend/internal/notification/providers/dingtalk.go`
  - 钉钉机器人 Webhook

- [ ] **Task 4.3.2**: 单元测试

### 4.4 高级规则引擎 [P1]

- [ ] **Task 4.4.1**: 实现比率规则
  - 支持分子/分母表达式
  - 计算百分比

- [ ] **Task 4.4.2**: 实现模式匹配规则
  - 使用正则表达式
  - 匹配日志字段

- [ ] **Task 4.4.3**: 实现复合条件规则
  - AND/OR 逻辑
  - 嵌套条件

- [ ] **Task 4.4.4**: 实现聚合规则
  - SUM, AVG, COUNT 等聚合函数

### 4.5 日志异常检测 [P1]

- [ ] **Task 4.5.1**: 实现错误率监控
  - 文件: `backend/internal/monitoring/error_rate.go`
  - 定时统计 4xx/5xx 比例
  - 触发 log.high_error_rate 事件

- [ ] **Task 4.5.2**: 实现可疑 IP 检测
  - 文件: `backend/internal/monitoring/suspicious_ip.go`
  - 检测异常访问模式
  - 触发 log.suspicious_ip 事件

- [ ] **Task 4.5.3**: 集成到日志采集流程
  - 文件: `backend/internal/ingest/caddy.go`
  - 实时检测和告警

### 4.6 通知优化 [P1]

- [ ] **Task 4.6.1**: 实现通知静默时间
  - 使用 Redis 记录上次通知时间
  - 避免告警风暴

- [ ] **Task 4.6.2**: 实现通知分组
  - 相同类型事件合并发送
  - 批量通知

- [ ] **Task 4.6.3**: 实现异步发送队列
  - 使用 channel 缓冲
  - 提高性能

---

## 阶段 5: 前端界面 (第 8 周)

### 5.1 通知渠道管理页面 [P1]

- [ ] **Task 5.1.1**: 创建渠道列表页面
  - 文件: `frontend/src/views/notification/channels/index.vue`
  - 列表展示、搜索、筛选

- [ ] **Task 5.1.2**: 创建渠道表单组件
  - 文件: `frontend/src/views/notification/channels/ChannelForm.vue`
  - 动态表单 (根据类型显示不同配置)

- [ ] **Task 5.1.3**: 创建测试通知对话框
  - 发送测试消息

### 5.2 告警规则管理页面 [P1]

- [ ] **Task 5.2.1**: 创建规则列表页面
  - 文件: `frontend/src/views/notification/rules/index.vue`

- [ ] **Task 5.2.2**: 创建规则表单组件
  - 文件: `frontend/src/views/notification/rules/RuleForm.vue`
  - 可视化规则编辑器

- [ ] **Task 5.2.3**: 实现规则启用/禁用切换

### 5.3 通知历史页面 [P1]

- [ ] **Task 5.3.1**: 创建历史列表页面
  - 文件: `frontend/src/views/notification/logs/index.vue`
  - 时间筛选、状态筛选

- [ ] **Task 5.3.2**: 创建详情抽屉
  - 查看通知详情
  - 错误信息展示

- [ ] **Task 5.3.3**: 实现重试功能

### 5.4 实时通知 [P2]

- [ ] **Task 5.4.1**: 实现 WebSocket 连接
  - 文件: `frontend/src/utils/websocket.ts`

- [ ] **Task 5.4.2**: 创建通知中心组件
  - 文件: `frontend/src/components/NotificationCenter.vue`
  - 实时展示通知

- [ ] **Task 5.4.3**: 集成到主布局

---

## 阶段 6: 测试与优化 (第 9 周)

### 6.1 单元测试 [P0]

- [ ] **Task 6.1.1**: Webhook Provider 测试
- [ ] **Task 6.1.2**: Email Provider 测试
- [ ] **Task 6.1.3**: Telegram Provider 测试
- [ ] **Task 6.1.4**: RuleEngine 测试
- [ ] **Task 6.1.5**: NotificationManager 测试
- [ ] **Task 6.1.6**: 测试覆盖率 > 80%

### 6.2 集成测试 [P1]

- [ ] **Task 6.2.1**: API 集成测试
  - 文件: `backend/test/integration/notification_test.go`

- [ ] **Task 6.2.2**: 端到端测试
  - 事件触发 -> 规则评估 -> 通知发送

### 6.3 性能测试 [P1]

- [ ] **Task 6.3.1**: 压力测试
  - 模拟大量事件
  - 测试通知发送性能

- [ ] **Task 6.3.2**: 性能优化
  - 优化数据库查询
  - 优化规则评估

### 6.4 文档 [P0]

- [ ] **Task 6.4.1**: 用户文档
  - 文件: `docs/notification-user-guide.md`
  - 配置指南、使用示例

- [ ] **Task 6.4.2**: 开发者文档
  - 文件: `docs/notification-developer-guide.md`
  - 架构说明、扩展指南

- [ ] **Task 6.4.3**: 部署文档
  - 更新 `deploy/README.md`
  - 通知配置说明

### 6.5 代码审查与优化 [P1]

- [ ] **Task 6.5.1**: 代码 Review
- [ ] **Task 6.5.2**: 错误处理优化
- [ ] **Task 6.5.3**: 日志完善
- [ ] **Task 6.5.4**: 性能优化

---

## 里程碑

- **M1 (Week 2)**: 基础设施完成,Webhook 通知可用
- **M2 (Week 4)**: Email、Telegram 通知可用,基础规则引擎完成
- **M3 (Week 5)**: 管理 API 完成
- **M4 (Week 7)**: 高级功能完成,日志异常检测可用
- **M5 (Week 8)**: 前端界面完成
- **M6 (Week 9)**: 测试完成,文档完成,功能上线

---

## 依赖关系

```
Task 1.1 -> Task 1.2 -> Task 1.7
Task 1.3 -> Task 1.4 -> Task 1.5
Task 1.6 -> Task 1.7
Task 1.7 -> Task 2.5
Task 2.3 -> Task 4.4
Task 3.1 -> Task 3.2, 3.3, 3.4
Task 3.2, 3.3 -> Task 5.1, 5.2
```

---

## 资源分配

- **后端开发**: 1-2 人
- **前端开发**: 1 人
- **测试**: 1 人 (兼职)
- **文档**: 1 人 (兼职)

---

## 风险与缓解

1. **风险**: 规则引擎复杂度高
   - **缓解**: 先实现简单规则,逐步扩展

2. **风险**: 通知风暴影响性能
   - **缓解**: 实现静默时间和速率限制

3. **风险**: 第三方 API 可用性
   - **缓解**: 实现重试机制和降级策略

4. **风险**: 时间估算不准确
   - **缓解**: 优先实现核心功能,可选功能后置

---

## 进度跟踪

使用 GitHub Issues 或 Jira 跟踪任务进度:
- 标签: notification, P0/P1/P2, phase-1/2/3/4/5/6
- 看板: To Do, In Progress, Review, Done
