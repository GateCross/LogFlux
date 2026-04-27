package model

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"
)

// CaddyLogQuery 是 Caddy 访问日志分页查询条件。
type CaddyLogQuery struct {
	Keyword  string
	Host     string
	Status   int
	Start    *time.Time
	End      *time.Time
	SortBy   string
	Order    string
	Page     int
	PageSize int
}

// SystemLogQuery 是系统日志分页查询条件。
type SystemLogQuery struct {
	Keyword  string
	Source   string
	Level    string
	Start    *time.Time
	End      *time.Time
	SortBy   string
	Order    string
	Page     int
	PageSize int
}

// DashboardTrendRow 是看板趋势聚合行。
type DashboardTrendRow struct {
	Bucket int64 `gorm:"column:bucket"`
	Count  int64 `gorm:"column:count"`
}

// DashboardGeoRow 是看板地域聚合行。
type DashboardGeoRow struct {
	Name  string `gorm:"column:name"`
	Value int64  `gorm:"column:value"`
}

type CaddyLogModel interface {
	List(ctx context.Context, query CaddyLogQuery) ([]CaddyLog, int64, error)
	CountRange(ctx context.Context, start, end time.Time) (int64, error)
	CountStatuses(ctx context.Context, start, end time.Time, statuses []int) (int64, error)
	CountStatusRange(ctx context.Context, start, end time.Time, min, max int) (int64, error)
	CountUniqueVisitor(ctx context.Context, start, end time.Time) (int64, error)
	CountUniqueRemoteIP(ctx context.Context, start, end time.Time) (int64, error)
	CountAttackIP(ctx context.Context, start, end time.Time, statuses []int) (int64, error)
	TrendRows(ctx context.Context, start, end time.Time, intervalSec int) ([]DashboardTrendRow, error)
	GeoRows(ctx context.Context, start, end time.Time, limit int) ([]DashboardGeoRow, error)
	Recent(ctx context.Context, start, end time.Time, limit int) ([]CaddyLog, error)
}

type SystemLogModel interface {
	List(ctx context.Context, query SystemLogQuery) ([]SystemLog, int64, error)
}

type defaultCaddyLogModel struct {
	db *gorm.DB
}

type defaultSystemLogModel struct {
	db *gorm.DB
}

func NewCaddyLogModel(db *gorm.DB) CaddyLogModel {
	return &defaultCaddyLogModel{db: db}
}

func NewSystemLogModel(db *gorm.DB) SystemLogModel {
	return &defaultSystemLogModel{db: db}
}

func caddyLogConn(db *gorm.DB, ctx context.Context) *gorm.DB {
	if ctx == nil {
		ctx = context.Background()
	}
	return db.WithContext(ctx)
}

func (m *defaultCaddyLogModel) List(ctx context.Context, query CaddyLogQuery) ([]CaddyLog, int64, error) {
	db := caddyLogConn(m.db, ctx).Model(&CaddyLog{})
	if keyword := strings.TrimSpace(query.Keyword); keyword != "" {
		like := "%" + keyword + "%"
		db = db.Where(
			"host ILIKE ? OR uri ILIKE ? OR remote_ip ILIKE ? OR client_ip ILIKE ?",
			like, like, like, like,
		)
	}
	if host := strings.TrimSpace(query.Host); host != "" {
		db = db.Where("host = ?", host)
	}
	if query.Status >= 0 {
		db = db.Where("status = ?", query.Status)
	}
	if query.Start != nil {
		db = db.Where("log_time >= ?", *query.Start)
	}
	if query.End != nil {
		db = db.Where("log_time <= ?", *query.End)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	page, pageSize := normalizePage(query.Page, query.PageSize)
	var logs []CaddyLog
	if err := db.Order(caddyLogOrder(query.SortBy, query.Order)).Limit(pageSize).Offset((page - 1) * pageSize).Find(&logs).Error; err != nil {
		return nil, 0, err
	}
	return logs, total, nil
}

func (m *defaultCaddyLogModel) CountRange(ctx context.Context, start, end time.Time) (int64, error) {
	var total int64
	err := m.baseRange(ctx, start, end).Count(&total).Error
	return total, err
}

func (m *defaultCaddyLogModel) CountStatuses(ctx context.Context, start, end time.Time, statuses []int) (int64, error) {
	var total int64
	err := m.baseRange(ctx, start, end).Where("status IN ?", statuses).Count(&total).Error
	return total, err
}

func (m *defaultCaddyLogModel) CountStatusRange(ctx context.Context, start, end time.Time, min, max int) (int64, error) {
	var total int64
	err := m.baseRange(ctx, start, end).Where("status >= ? AND status < ?", min, max).Count(&total).Error
	return total, err
}

func (m *defaultCaddyLogModel) CountUniqueVisitor(ctx context.Context, start, end time.Time) (int64, error) {
	var total int64
	err := caddyLogConn(m.db, ctx).Raw(
		`SELECT COUNT(DISTINCT COALESCE(NULLIF(client_ip, ''), remote_ip))
		 FROM caddy_logs
		 WHERE log_time BETWEEN ? AND ? AND (client_ip <> '' OR remote_ip <> '')`,
		start, end,
	).Scan(&total).Error
	return total, err
}

func (m *defaultCaddyLogModel) CountUniqueRemoteIP(ctx context.Context, start, end time.Time) (int64, error) {
	var total int64
	err := caddyLogConn(m.db, ctx).Raw(
		"SELECT COUNT(DISTINCT remote_ip) FROM caddy_logs WHERE log_time BETWEEN ? AND ? AND remote_ip <> ''",
		start, end,
	).Scan(&total).Error
	return total, err
}

func (m *defaultCaddyLogModel) CountAttackIP(ctx context.Context, start, end time.Time, statuses []int) (int64, error) {
	var total int64
	err := caddyLogConn(m.db, ctx).Raw(
		"SELECT COUNT(DISTINCT remote_ip) FROM caddy_logs WHERE log_time BETWEEN ? AND ? AND status IN ? AND remote_ip <> ''",
		start, end, statuses,
	).Scan(&total).Error
	return total, err
}

func (m *defaultCaddyLogModel) TrendRows(ctx context.Context, start, end time.Time, intervalSec int) ([]DashboardTrendRow, error) {
	rows := make([]DashboardTrendRow, 0)
	err := caddyLogConn(m.db, ctx).Raw(
		`SELECT floor(extract(epoch from log_time) / ?) * ? AS bucket, COUNT(*) AS count
		 FROM caddy_logs
		 WHERE log_time BETWEEN ? AND ?
		 GROUP BY bucket
		 ORDER BY bucket`,
		intervalSec, intervalSec, start, end,
	).Scan(&rows).Error
	return rows, err
}

func (m *defaultCaddyLogModel) GeoRows(ctx context.Context, start, end time.Time, limit int) ([]DashboardGeoRow, error) {
	rows := make([]DashboardGeoRow, 0)
	err := caddyLogConn(m.db, ctx).Raw(
		`SELECT COALESCE(NULLIF(country, ''), '未知') AS name, COUNT(*) AS value
		 FROM caddy_logs
		 WHERE log_time BETWEEN ? AND ?
		 GROUP BY name
		 ORDER BY value DESC
		 LIMIT ?`,
		start, end, limit,
	).Scan(&rows).Error
	return rows, err
}

func (m *defaultCaddyLogModel) Recent(ctx context.Context, start, end time.Time, limit int) ([]CaddyLog, error) {
	var logs []CaddyLog
	err := caddyLogConn(m.db, ctx).Model(&CaddyLog{}).
		Where("log_time >= ? AND log_time <= ?", start, end).
		Order("log_time desc, id desc").
		Limit(limit).
		Find(&logs).Error
	return logs, err
}

func (m *defaultCaddyLogModel) baseRange(ctx context.Context, start, end time.Time) *gorm.DB {
	return caddyLogConn(m.db, ctx).Model(&CaddyLog{}).Where("log_time >= ? AND log_time <= ?", start, end)
}

func (m *defaultSystemLogModel) List(ctx context.Context, query SystemLogQuery) ([]SystemLog, int64, error) {
	db := caddyLogConn(m.db, ctx).Model(&SystemLog{})
	if keyword := strings.TrimSpace(query.Keyword); keyword != "" {
		like := "%" + keyword + "%"
		db = db.Where("message ILIKE ? OR caller ILIKE ? OR raw_log ILIKE ?", like, like, like)
	}
	if source := strings.TrimSpace(query.Source); source != "" {
		db = db.Where("source = ?", source)
	}
	if level := strings.TrimSpace(strings.ToLower(query.Level)); level != "" {
		db = db.Where("level = ?", level)
	}
	if query.Start != nil {
		db = db.Where("log_time >= ?", *query.Start)
	}
	if query.End != nil {
		db = db.Where("log_time <= ?", *query.End)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	page, pageSize := normalizePage(query.Page, query.PageSize)
	var logs []SystemLog
	if err := db.Order(caddyLogOrder(query.SortBy, query.Order)).Limit(pageSize).Offset((page - 1) * pageSize).Find(&logs).Error; err != nil {
		return nil, 0, err
	}
	return logs, total, nil
}

func caddyLogOrder(sortBy, order string) string {
	if strings.ToLower(strings.TrimSpace(sortBy)) == "logtime" ||
		strings.ToLower(strings.TrimSpace(sortBy)) == "log_time" ||
		strings.ToLower(strings.TrimSpace(sortBy)) == "time" {
		if strings.ToLower(strings.TrimSpace(order)) == "asc" {
			return "log_time asc, id asc"
		}
	}
	return "log_time desc, id desc"
}

func normalizePage(page, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	return page, pageSize
}
