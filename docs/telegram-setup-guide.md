# Telegram 通知配置示例

## 1. 获取 Bot Token

1. 在 Telegram 中找到 @BotFather
2. 发送 `/newbot` 创建新机器人
3. 按照提示设置机器人名称
4. 获取 Bot Token (格式: `123456789:ABCdefGHIjklMNOpqrsTUVwxyz`)

## 2. 获取 Chat ID

### 方法 1: 使用个人聊天
1. 向你的机器人发送任意消息
2. 访问 `https://api.telegram.org/bot<YOUR_BOT_TOKEN>/getUpdates`
3. 在返回的 JSON 中找到 `chat.id`

### 方法 2: 使用群组聊天
1. 将机器人添加到群组
2. 在群组中发送任意消息
3. 访问 `https://api.telegram.org/bot<YOUR_BOT_TOKEN>/getUpdates`
4. 在返回的 JSON 中找到 `chat.id` (负数表示群组)

## 3. 配置渠道

渠道**只能**在 Web 界面创建：`通知 → 通知渠道 → 新增`。填「类型 = Telegram」，配置项：

| 字段 | 说明 |
|---|---|
| `bot_token` | BotFather 给的 Token |
| `chat_id` | 个人聊天为正数，群组为负数 |

> 不存在通过 `backend/etc/config.yaml` 配置渠道的方式。配置结构体里虽然有 `Notification.Channels` 字段，但运行时**从不读取**——渠道一律从 `notification_channels` 表加载。手写该段配置不会生效。

创建后可在通知规则里按事件类型订阅，见本文第 7 节。

## 4. 测试通知

重启后端服务后,系统会自动发送启动通知:

```bash
cd backend
./logflux-api -f etc/config.yaml
```

你应该会在 Telegram 中收到类似这样的消息:

```
*系统启动*

ℹ️ *级别:* info
🕒 *时间:* 2026-01-29 12:00:00

📝 *消息:*
LogFlux 通知系统已启动
```

## 5. 消息格式

Telegram 提供者使用 Markdown V2 格式,支持:
- **粗体文本**
- _斜体文本_
- `代码`
- ```代码块```
- 表情符号

### 级别图标映射
- `info` → ℹ️
- `warning` → ⚠️
- `error` → ❌
- `critical` → 🚨
- `success` → ✅

## 6. 常见问题

### Q: Bot 收不到消息?
A:
1. 检查 Bot Token 是否正确
2. 确保向机器人发送过至少一条消息
3. 检查 Chat ID 是否正确

### Q: 消息格式乱码?
A: Telegram 使用 Markdown V2 格式,特殊字符会自动转义

### Q: 如何发送到多个群组?
A: 创建多个 channel,每个配置不同的 chat_id

## 7. 高级用法

### 7.1 创建专属告警群组
1. 创建新群组
2. 添加机器人为管理员
3. 获取群组 Chat ID (负数)
4. 配置多个渠道分别推送不同级别的消息

### 7.2 事件订阅写法

在渠道的「事件」字段里填匹配模式，支持三种写法：

| 写法 | 含义 |
|---|---|
| `system.startup` | 精确匹配 |
| `system.*` | 前缀通配，匹配 `system.` 开头的所有事件 |
| `*` | 全部事件 |

常用组合（下表「真正会触发」的事件才有效）：

```
security.waf_source_sync_failed
security.waf_release_activate_failed
cert.*
system.startup
```

### 7.3 真正会触发通知的事件

事件常量在 `internal/notification/event.go` 里定义了 29 个，但**只有下面 11 个有发射点**，订阅其他事件收不到消息：

| 事件 | 触发场景 |
|---|---|
| `system.startup` | 后端启动 |
| `cert.expiring` | 证书临近到期（14/7 天 warning，1 天 critical） |
| `cert.expired` | 证书已过期 |
| `security.waf_source_check_success` | WAF 源检查到新版本 |
| `security.waf_source_check_failed` | WAF 源检查失败 |
| `security.waf_source_sync_success` | WAF 源同步成功 |
| `security.waf_source_sync_failed` | WAF 源同步失败 |
| `security.waf_release_activate_success` | CRS 版本激活成功 |
| `security.waf_release_activate_failed` | CRS 版本激活失败 |
| `archive.failed` | 日志清理任务失败（复用归档事件名） |
| `archive.completed` | 日志清理任务完成（复用归档事件名） |

推荐的告警组合：

```
security.waf_source_sync_failed
security.waf_release_activate_failed
cert.*
```

> **不要在意图上依赖这些不生效的事件**：`log.*`（5 个）、`caddy.*`（3 个）、`security.login_failed` / `brute_force` / `admin_login` / `permission_denied`、`system.error` / `system.shutdown`、`database.connection_failed`、`redis.connection_failed`、`archive.slow` / `anomaly` 均无发射点。它们仍出现在通知规则的事件下拉里（`get_event_list_logic.go` 静态注册），选中后规则永远不会命中。
>
> 另外 `error.*` / `critical.*` 是**级别**不是前缀，写了也匹配不到。

## 8. 配置验证

保存渠道时后端会校验 `bot_token` 与 `chat_id`：

- `bot_token` 必填
- `chat_id` 必填且必须是整数

校验失败时接口返回中文错误，渠道不会被创建。发送失败会记入「通知 → 发送日志」，可按渠道与时间排查。
