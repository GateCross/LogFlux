package waf

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// WafEvent 是 Coraza 审计日志的归一化事件，一次规则命中一行；
// 同一事务的多条命中通过 TransactionID 关联，
// (transaction_id, rule_id) 唯一索引保证重复采集与回填幂等。
type WafEvent struct {
	ID        uint      `gorm:"primarykey"`
	CreatedAt time.Time `gorm:"index"`
	EventTime time.Time `gorm:"index:idx_waf_event_time;not null"`

	TransactionID string `gorm:"column:transaction_id;size:64;not null;uniqueIndex:uk_waf_event_tx_rule,priority:1"`
	RuleID        string `gorm:"size:32;not null;default:'';uniqueIndex:uk_waf_event_tx_rule,priority:2;index"`
	RuleMsg       string `gorm:"size:512"`
	Severity      string `gorm:"size:16;index"`
	Interrupted   bool   `gorm:"not null;default:false;index"`
	Status        int    `gorm:"not null;default:0"`

	ClientIP string `gorm:"size:64;index"`
	Country  string `gorm:"size:100;index"`
	Province string `gorm:"size:100"`
	City     string `gorm:"size:100"`

	Host      string `gorm:"size:255;index"`
	Method    string `gorm:"size:10"`
	Uri       string `gorm:"type:text"`
	UserAgent string `gorm:"type:text"`

	// 来源访问日志 ID，回溯 RawLog 用；0 表示来源行已不可追溯
	CaddyLogID uint   `gorm:"not null;default:0"`
	RawJSON    string `gorm:"type:jsonb;comment:审计日志原始 JSON"`
}

// TableName 返回表名
func (WafEvent) TableName() string {
	return "waf_events"
}

// WafEventQuery 事件列表查询条件；时间为 nil 表示不限
type WafEventQuery struct {
	Page      int
	PageSize  int
	StartTime *time.Time
	EndTime   *time.Time
	ClientIP  string
	Host      string
	RuleID    string
	Severity  string
	Action    string // block | detect | 空 = 全部
	Keyword   string // 模糊匹配 URI 与规则消息
}

// WafEventTrendRow 趋势桶，Interrupted 区分拦截与仅检测两条序列
type WafEventTrendRow struct {
	Bucket      int64 `gorm:"column:bucket"`
	Interrupted bool  `gorm:"column:interrupted"`
	Count       int64 `gorm:"column:count"`
}

// WafEventCountRow Top 榜通用计数行
type WafEventCountRow struct {
	Name  string `gorm:"column:name"`
	Count int64  `gorm:"column:count"`
}

// WafEventGeoRow 攻击源地理分布行
type WafEventGeoRow struct {
	Name  string `gorm:"column:name"`
	Value int64  `gorm:"column:value"`
}

type WafEventModel interface {
	SaveIgnoreConflict(ctx context.Context, events []*WafEvent) error
	List(ctx context.Context, query WafEventQuery) ([]WafEvent, int64, error)
	Count(ctx context.Context, start, end time.Time) (total int64, blocked int64, err error)
	CountUniqueIP(ctx context.Context, start, end time.Time) (int64, error)
	TrendRows(ctx context.Context, start, end time.Time, intervalSec int) ([]WafEventTrendRow, error)
	TopRules(ctx context.Context, start, end time.Time, limit int) ([]WafEventCountRow, error)
	TopIPs(ctx context.Context, start, end time.Time, limit int) ([]WafEventCountRow, error)
	TopHosts(ctx context.Context, start, end time.Time, limit int) ([]WafEventCountRow, error)
	CountryRows(ctx context.Context, start, end time.Time, limit int) ([]WafEventGeoRow, error)
}

type defaultWafEventModel struct {
	db *gorm.DB
}

func NewWafEventModel(db *gorm.DB) WafEventModel {
	return &defaultWafEventModel{db: db}
}

func (m *defaultWafEventModel) conn(ctx context.Context) *gorm.DB {
	if ctx == nil {
		ctx = context.Background()
	}
	return m.db.WithContext(ctx)
}

// SaveIgnoreConflict 批量写入，(transaction_id, rule_id) 冲突时跳过，
// 供采集双写与历史回填共用。
func (m *defaultWafEventModel) SaveIgnoreConflict(ctx context.Context, events []*WafEvent) error {
	if len(events) == 0 {
		return nil
	}
	return m.conn(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "transaction_id"}, {Name: "rule_id"}},
		DoNothing: true,
	}).CreateInBatches(events, 200).Error
}

func (m *defaultWafEventModel) filter(ctx context.Context, query WafEventQuery) *gorm.DB {
	db := m.conn(ctx).Model(&WafEvent{})
	if query.StartTime != nil {
		db = db.Where("event_time >= ?", *query.StartTime)
	}
	if query.EndTime != nil {
		db = db.Where("event_time <= ?", *query.EndTime)
	}
	if ip := strings.TrimSpace(query.ClientIP); ip != "" {
		db = db.Where("client_ip = ?", ip)
	}
	if host := strings.TrimSpace(query.Host); host != "" {
		db = db.Where("host ILIKE ?", "%"+host+"%")
	}
	if ruleID := strings.TrimSpace(query.RuleID); ruleID != "" {
		db = db.Where("rule_id = ?", ruleID)
	}
	if severity := strings.TrimSpace(query.Severity); severity != "" {
		db = db.Where("severity = ?", severity)
	}
	switch strings.ToLower(strings.TrimSpace(query.Action)) {
	case "block":
		db = db.Where("interrupted = ?", true)
	case "detect":
		db = db.Where("interrupted = ?", false)
	}
	if keyword := strings.TrimSpace(query.Keyword); keyword != "" {
		db = db.Where("(uri ILIKE ? OR rule_msg ILIKE ?)", "%"+keyword+"%", "%"+keyword+"%")
	}
	return db
}

func (m *defaultWafEventModel) List(ctx context.Context, query WafEventQuery) ([]WafEvent, int64, error) {
	page, pageSize := normalizeWafEventPage(query.Page, query.PageSize)

	db := m.filter(ctx, query)
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var events []WafEvent
	offset := (page - 1) * pageSize
	if err := db.Order("event_time desc, id desc").Limit(pageSize).Offset(offset).Find(&events).Error; err != nil {
		return nil, 0, err
	}
	return events, total, nil
}

func (m *defaultWafEventModel) Count(ctx context.Context, start, end time.Time) (int64, int64, error) {
	var rows []struct {
		Interrupted bool  `gorm:"column:interrupted"`
		Count       int64 `gorm:"column:count"`
	}
	err := m.conn(ctx).Model(&WafEvent{}).
		Select("interrupted, COUNT(*) AS count").
		Where("event_time >= ? AND event_time <= ?", start, end).
		Group("interrupted").
		Scan(&rows).Error
	if err != nil {
		return 0, 0, err
	}

	var total, blocked int64
	for _, row := range rows {
		total += row.Count
		if row.Interrupted {
			blocked = row.Count
		}
	}
	return total, blocked, nil
}

func (m *defaultWafEventModel) CountUniqueIP(ctx context.Context, start, end time.Time) (int64, error) {
	var total int64
	err := m.conn(ctx).Raw(
		`SELECT COUNT(DISTINCT client_ip)
		 FROM waf_events
		 WHERE event_time BETWEEN ? AND ? AND client_ip <> ''`,
		start, end,
	).Scan(&total).Error
	return total, err
}

func (m *defaultWafEventModel) TrendRows(ctx context.Context, start, end time.Time, intervalSec int) ([]WafEventTrendRow, error) {
	rows := make([]WafEventTrendRow, 0)
	err := m.conn(ctx).Raw(
		`SELECT floor(extract(epoch from event_time) / ?) * ? AS bucket, interrupted, COUNT(*) AS count
		 FROM waf_events
		 WHERE event_time BETWEEN ? AND ?
		 GROUP BY bucket, interrupted
		 ORDER BY bucket`,
		intervalSec, intervalSec, start, end,
	).Scan(&rows).Error
	return rows, err
}

func (m *defaultWafEventModel) TopRules(ctx context.Context, start, end time.Time, limit int) ([]WafEventCountRow, error) {
	rows := make([]WafEventCountRow, 0)
	err := m.conn(ctx).Raw(
		`SELECT COALESCE(NULLIF(rule_id, ''), '无规则') || ' ' || COALESCE(NULLIF(rule_msg, ''), '') AS name, COUNT(*) AS count
		 FROM waf_events
		 WHERE event_time BETWEEN ? AND ?
		 GROUP BY rule_id, rule_msg
		 ORDER BY count DESC
		 LIMIT ?`,
		start, end, limit,
	).Scan(&rows).Error
	return rows, err
}

func (m *defaultWafEventModel) TopIPs(ctx context.Context, start, end time.Time, limit int) ([]WafEventCountRow, error) {
	rows := make([]WafEventCountRow, 0)
	err := m.conn(ctx).Raw(
		`SELECT COALESCE(NULLIF(client_ip, ''), '未知') AS name, COUNT(*) AS count
		 FROM waf_events
		 WHERE event_time BETWEEN ? AND ?
		 GROUP BY name
		 ORDER BY count DESC
		 LIMIT ?`,
		start, end, limit,
	).Scan(&rows).Error
	return rows, err
}

func (m *defaultWafEventModel) TopHosts(ctx context.Context, start, end time.Time, limit int) ([]WafEventCountRow, error) {
	rows := make([]WafEventCountRow, 0)
	err := m.conn(ctx).Raw(
		`SELECT COALESCE(NULLIF(host, ''), '未知') AS name, COUNT(*) AS count
		 FROM waf_events
		 WHERE event_time BETWEEN ? AND ?
		 GROUP BY name
		 ORDER BY count DESC
		 LIMIT ?`,
		start, end, limit,
	).Scan(&rows).Error
	return rows, err
}

func (m *defaultWafEventModel) CountryRows(ctx context.Context, start, end time.Time, limit int) ([]WafEventGeoRow, error) {
	rows := make([]WafEventGeoRow, 0)
	err := m.conn(ctx).Raw(
		`SELECT COALESCE(NULLIF(country, ''), '未知') AS name, COUNT(*) AS value
		 FROM waf_events
		 WHERE event_time BETWEEN ? AND ?
		 GROUP BY name
		 ORDER BY value DESC
		 LIMIT ?`,
		start, end, limit,
	).Scan(&rows).Error
	return rows, err
}

func normalizeWafEventPage(page, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}
	return page, pageSize
}
