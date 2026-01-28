-- 性能优化迁移脚本
-- 添加复合索引和优化查询性能

-- 1. 为 caddy_logs 添加复合索引
CREATE INDEX IF NOT EXISTS idx_caddy_logs_log_time_status ON caddy_logs(log_time DESC, status);
CREATE INDEX IF NOT EXISTS idx_caddy_logs_remote_ip_log_time ON caddy_logs(remote_ip, log_time DESC);
CREATE INDEX IF NOT EXISTS idx_caddy_logs_host_log_time ON caddy_logs(host, log_time DESC);
CREATE INDEX IF NOT EXISTS idx_caddy_logs_status_log_time ON caddy_logs(status, log_time DESC);

-- 2. 为全文搜索创建 GIN 索引（可选，如果需要更高级的搜索）
-- CREATE INDEX IF NOT EXISTS idx_caddy_logs_uri_gin ON caddy_logs USING gin(to_tsvector('simple', uri));

-- 3. 创建归档表（与主表结构相同）
CREATE TABLE IF NOT EXISTS caddy_logs_archive (
    LIKE caddy_logs INCLUDING ALL
);

-- 4. 为归档表添加索引
CREATE INDEX IF NOT EXISTS idx_archive_log_time ON caddy_logs_archive(log_time DESC);
CREATE INDEX IF NOT EXISTS idx_archive_status ON caddy_logs_archive(status);

-- 5. 创建分区表（PostgreSQL 10+）
-- 注意：如果表已经存在数据，需要先迁移数据再转换为分区表
-- 这里提供新建分区表的示例，实际使用时需要根据情况调整

/*
-- 创建分区主表（如果是全新部署）
CREATE TABLE IF NOT EXISTS caddy_logs_partitioned (
    id BIGSERIAL,
    created_at TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE,
    log_time TIMESTAMP WITH TIME ZONE NOT NULL,
    country VARCHAR(100),
    province VARCHAR(100),
    city VARCHAR(100),
    host VARCHAR(255),
    method VARCHAR(10),
    uri TEXT,
    proto VARCHAR(20),
    status INTEGER,
    size BIGINT,
    user_agent TEXT,
    remote_ip VARCHAR(50),
    client_ip VARCHAR(50),
    raw_log JSONB,
    extra_data JSONB,
    PRIMARY KEY (id, log_time)
) PARTITION BY RANGE (log_time);

-- 创建按月分区的示例
CREATE TABLE caddy_logs_y2026m01 PARTITION OF caddy_logs_partitioned
    FOR VALUES FROM ('2026-01-01') TO ('2026-02-01');

CREATE TABLE caddy_logs_y2026m02 PARTITION OF caddy_logs_partitioned
    FOR VALUES FROM ('2026-02-01') TO ('2026-03-01');
*/

-- 6. 创建自动归档函数
CREATE OR REPLACE FUNCTION archive_old_logs(retention_days INTEGER DEFAULT 90)
RETURNS INTEGER AS $$
DECLARE
    archived_count INTEGER;
    archive_date TIMESTAMP;
BEGIN
    archive_date := NOW() - (retention_days || ' days')::INTERVAL;

    -- 将旧数据移动到归档表
    WITH moved_rows AS (
        DELETE FROM caddy_logs
        WHERE log_time < archive_date
        RETURNING *
    )
    INSERT INTO caddy_logs_archive
    SELECT * FROM moved_rows;

    GET DIAGNOSTICS archived_count = ROW_COUNT;

    RETURN archived_count;
END;
$$ LANGUAGE plpgsql;

-- 使用示例:
-- SELECT archive_old_logs(90); -- 归档 90 天前的日志
