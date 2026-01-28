-- 通知功能数据库表结构
-- 创建时间: 2026-01-28

-- 1. 通知渠道表
CREATE TABLE IF NOT EXISTS notification_channels (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    -- 基本信息
    name VARCHAR(100) NOT NULL UNIQUE,
    type VARCHAR(50) NOT NULL,  -- webhook, email, telegram, slack, wecom, dingtalk
    enabled BOOLEAN NOT NULL DEFAULT TRUE,

    -- 配置 (JSON)
    -- Webhook: {"url": "...", "method": "POST", "headers": {...}}
    -- Email: {"smtp_host": "...", "smtp_port": 587, "username": "...", "password": "...", "from": "...", "to": [...]}
    -- Telegram: {"bot_token": "...", "chat_id": "..."}
    -- Slack: {"webhook_url": "..."}
    -- WeCom: {"webhook_url": "..."}
    -- DingTalk: {"webhook_url": "...", "secret": "..."}
    config JSONB NOT NULL,

    -- 事件过滤 (订阅的事件类型)
    -- 支持通配符,如 "system.*", "log.high_error_rate"
    events TEXT[] NOT NULL DEFAULT '{}',

    -- 描述
    description TEXT
);

-- 索引
CREATE INDEX IF NOT EXISTS idx_notification_channels_type ON notification_channels(type);
CREATE INDEX IF NOT EXISTS idx_notification_channels_enabled ON notification_channels(enabled);

-- 注释
COMMENT ON TABLE notification_channels IS '通知渠道配置表';
COMMENT ON COLUMN notification_channels.name IS '渠道名称 (唯一)';
COMMENT ON COLUMN notification_channels.type IS '渠道类型: webhook, email, telegram, slack, wecom, dingtalk';
COMMENT ON COLUMN notification_channels.enabled IS '是否启用';
COMMENT ON COLUMN notification_channels.config IS '渠道配置 (JSONB)';
COMMENT ON COLUMN notification_channels.events IS '订阅的事件类型 (支持通配符)';

-- 2. 告警规则表
CREATE TABLE IF NOT EXISTS notification_rules (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    -- 基本信息
    name VARCHAR(100) NOT NULL UNIQUE,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,

    -- 规则类型: threshold, frequency, ratio, pattern, composite
    rule_type VARCHAR(50) NOT NULL,

    -- 条件表达式 (JSON)
    -- Threshold: {"field": "status", "operator": ">=", "value": 500}
    -- Frequency: {"event": "security.login_failed", "count": 5, "window": "1m", "group_by": "remote_ip"}
    -- Ratio: {"numerator": "status >= 500", "denominator": "total", "threshold": 0.05, "window": "5m"}
    -- Pattern: {"field": "uri", "pattern": "^/admin/.*"}
    -- Composite: {"operator": "AND", "conditions": [...]}
    condition JSONB NOT NULL,

    -- 触发事件类型
    event_type VARCHAR(100) NOT NULL,

    -- 通知渠道 ID (关联 notification_channels)
    channel_ids BIGINT[] NOT NULL DEFAULT '{}',

    -- 通知模板 (可选,如果为空则使用默认模板)
    template TEXT,

    -- 静默时间 (秒) - 避免告警风暴
    silence_duration INTEGER DEFAULT 300,

    -- 最后触发时间
    last_triggered_at TIMESTAMP,

    -- 触发次数
    trigger_count INTEGER DEFAULT 0,

    -- 描述
    description TEXT
);

-- 索引
CREATE INDEX IF NOT EXISTS idx_notification_rules_event_type ON notification_rules(event_type);
CREATE INDEX IF NOT EXISTS idx_notification_rules_enabled ON notification_rules(enabled);
CREATE INDEX IF NOT EXISTS idx_notification_rules_last_triggered ON notification_rules(last_triggered_at);

-- 注释
COMMENT ON TABLE notification_rules IS '告警规则表';
COMMENT ON COLUMN notification_rules.name IS '规则名称 (唯一)';
COMMENT ON COLUMN notification_rules.rule_type IS '规则类型: threshold, frequency, ratio, pattern, composite';
COMMENT ON COLUMN notification_rules.condition IS '条件表达式 (JSONB)';
COMMENT ON COLUMN notification_rules.event_type IS '触发事件类型';
COMMENT ON COLUMN notification_rules.channel_ids IS '通知渠道 ID 数组';
COMMENT ON COLUMN notification_rules.template IS '自定义通知模板';
COMMENT ON COLUMN notification_rules.silence_duration IS '静默时间 (秒)';

-- 3. 通知历史表
CREATE TABLE IF NOT EXISTS notification_logs (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),

    -- 关联的渠道和规则
    channel_id BIGINT NOT NULL,
    rule_id BIGINT,  -- 可选,手动发送的通知没有规则

    -- 事件信息
    event_type VARCHAR(100) NOT NULL,
    event_data JSONB,  -- 事件详细数据

    -- 发送状态: pending, success, failed
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    error_message TEXT,

    -- 发送时间
    sent_at TIMESTAMP
);

-- 索引
CREATE INDEX IF NOT EXISTS idx_notification_logs_channel_id ON notification_logs(channel_id);
CREATE INDEX IF NOT EXISTS idx_notification_logs_rule_id ON notification_logs(rule_id);
CREATE INDEX IF NOT EXISTS idx_notification_logs_event_type ON notification_logs(event_type);
CREATE INDEX IF NOT EXISTS idx_notification_logs_status ON notification_logs(status);
CREATE INDEX IF NOT EXISTS idx_notification_logs_created_at ON notification_logs(created_at DESC);

-- 外键约束
ALTER TABLE notification_logs
    ADD CONSTRAINT fk_notification_logs_channel
    FOREIGN KEY (channel_id)
    REFERENCES notification_channels(id)
    ON DELETE CASCADE;

ALTER TABLE notification_logs
    ADD CONSTRAINT fk_notification_logs_rule
    FOREIGN KEY (rule_id)
    REFERENCES notification_rules(id)
    ON DELETE SET NULL;

-- 注释
COMMENT ON TABLE notification_logs IS '通知历史记录表';
COMMENT ON COLUMN notification_logs.channel_id IS '通知渠道 ID';
COMMENT ON COLUMN notification_logs.rule_id IS '触发规则 ID (可选)';
COMMENT ON COLUMN notification_logs.event_type IS '事件类型';
COMMENT ON COLUMN notification_logs.event_data IS '事件数据 (JSONB)';
COMMENT ON COLUMN notification_logs.status IS '发送状态: pending, success, failed';
COMMENT ON COLUMN notification_logs.sent_at IS '实际发送时间';

-- 4. 创建自动更新 updated_at 的触发器函数
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 为 notification_channels 表添加触发器
CREATE TRIGGER update_notification_channels_updated_at
    BEFORE UPDATE ON notification_channels
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 为 notification_rules 表添加触发器
CREATE TRIGGER update_notification_rules_updated_at
    BEFORE UPDATE ON notification_rules
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 5. 插入示例数据 (可选,用于测试)
-- INSERT INTO notification_channels (name, type, enabled, config, events) VALUES
-- ('webhook-test', 'webhook', true,
--  '{"url": "https://webhook.site/your-unique-url", "method": "POST", "headers": {"Content-Type": "application/json"}}',
--  ARRAY['system.*', 'archive.failed']);

-- INSERT INTO notification_rules (name, enabled, rule_type, event_type, condition, channel_ids) VALUES
-- ('归档任务失败告警', true, 'threshold', 'archive.failed',
--  '{"field": "error_count", "operator": ">=", "value": 1}',
--  ARRAY[1]);
