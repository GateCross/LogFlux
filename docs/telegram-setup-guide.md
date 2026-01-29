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

## 3. 配置示例

### 方法 A: 通过配置文件

在 `backend/etc/config.yaml` 中添加:

```yaml
notification:
  channels:
    - name: "telegram-alerts"
      type: "telegram"
      enabled: true
      description: "Telegram 告警通知"
      events:
        - "system.*"
        - "error.*"
        - "critical.*"
      config:
        bot_token: "123456789:ABCdefGHIjklMNOpqrsTUVwxyz"
        chat_id: "123456789"  # 你的 chat_id
```

### 方法 B: 通过数据库直接插入

```sql
INSERT INTO notification_channels (name, type, enabled, description, config, events, created_at, updated_at)
VALUES (
  'telegram-alerts',
  'telegram',
  true,
  'Telegram 告警通知',
  '{"bot_token": "123456789:ABCdefGHIjklMNOpqrsTUVwxyz", "chat_id": "123456789"}',
  ARRAY['system.*', 'error.*', 'critical.*'],
  NOW(),
  NOW()
);
```

## 4. 测试通知

重启后端服务后,系统会自动发送启动通知:

```bash
cd backend
./logflux -f etc/config.yaml
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

### 7.2 事件过滤示例

```yaml
# 仅接收系统事件
events:
  - "system.*"

# 仅接收错误和严重事件
events:
  - "error.*"
  - "critical.*"

# 接收所有事件
events:
  - "*"
```

## 8. 配置验证

在代码中,Telegram 配置会自动验证:
- ✅ `bot_token` 必填
- ✅ `chat_id` 必填且必须是数字
- ✅ 配置格式正确

验证失败会在日志中输出错误信息。
