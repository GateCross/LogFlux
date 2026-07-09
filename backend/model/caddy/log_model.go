package caddy

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

// 按 host 聚合的状态计数行。
// 仅返回时间窗内存在匹配日志的 host；无日志 host 不出现在结果中，
// 由调用方（如站点近窗指标 API）对请求的 hosts 补 0。
type HostStatusCount struct {
	Host  string `gorm:"column:host"`
	Count int64  `gorm:"column:count"`
}

type CaddyLogModel interface {
	Create(ctx context.Context, log *CaddyLog) error
	List(ctx context.Context, query CaddyLogQuery) ([]CaddyLog, int64, error)
	Clear(ctx context.Context) error
	CountRange(ctx context.Context, start, end time.Time) (int64, error)
	CountStatuses(ctx context.Context, start, end time.Time, statuses []int) (int64, error)
	CountStatusRange(ctx context.Context, start, end time.Time, min, max int) (int64, error)
	// 在时间窗内按 host 批量聚合指定状态范围（min 含，max 不含）的计数。
	// hosts 为空时返回空切片；结果仅含有匹配行的 host。
	CountStatusByHosts(ctx context.Context, start, end time.Time, hosts []string, min, max int) ([]HostStatusCount, error)
	CountUniqueVisitor(ctx context.Context, start, end time.Time) (int64, error)
	CountUniqueRemoteIP(ctx context.Context, start, end time.Time) (int64, error)
	CountAttackIP(ctx context.Context, start, end time.Time, statuses []int) (int64, error)
	TrendRows(ctx context.Context, start, end time.Time, intervalSec int) ([]DashboardTrendRow, error)
	GeoRows(ctx context.Context, start, end time.Time, limit int) ([]DashboardGeoRow, error)
	ProvinceRows(ctx context.Context, start, end time.Time, limit int) ([]DashboardGeoRow, error)
	Recent(ctx context.Context, start, end time.Time, limit int) ([]CaddyLog, error)
}

type defaultCaddyLogModel struct {
	db *gorm.DB
}

func NewCaddyLogModel(db *gorm.DB) CaddyLogModel {
	return &defaultCaddyLogModel{db: db}
}

func (m *defaultCaddyLogModel) Create(ctx context.Context, log *CaddyLog) error {
	return conn(m.db, ctx).Create(log).Error
}

func conn(db *gorm.DB, ctx context.Context) *gorm.DB {
	if ctx == nil {
		ctx = context.Background()
	}
	return db.WithContext(ctx)
}

func (m *defaultCaddyLogModel) List(ctx context.Context, query CaddyLogQuery) ([]CaddyLog, int64, error) {
	db := conn(m.db, ctx).Model(&CaddyLog{})
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
	if err := db.Order(logOrder(query.SortBy, query.Order)).Limit(pageSize).Offset((page - 1) * pageSize).Find(&logs).Error; err != nil {
		return nil, 0, err
	}
	return logs, total, nil
}

func (m *defaultCaddyLogModel) Clear(ctx context.Context) error {
	return conn(m.db, ctx).Exec(`TRUNCATE TABLE "caddy_logs" RESTART IDENTITY`).Error
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

// 按 host 批量统计时间窗内状态范围计数。
// 状态语义与按范围计数一致：status ∈ [min, max)。
// 例如 4xx 传 min=400,max=500；5xx 传 min=500,max=600。
// 空 hosts 直接返回空结果；无匹配日志的 host 不会出现在返回值中。
func (m *defaultCaddyLogModel) CountStatusByHosts(ctx context.Context, start, end time.Time, hosts []string, min, max int) ([]HostStatusCount, error) {
	rows := make([]HostStatusCount, 0)
	if len(hosts) == 0 {
		return rows, nil
	}
	// 利用 host / status / log_time 相关索引：时间窗 + host IN + status 范围 + GROUP BY host
	err := m.baseRange(ctx, start, end).
		Select("host, COUNT(*) AS count").
		Where("host IN ?", hosts).
		Where("status >= ? AND status < ?", min, max).
		Group("host").
		Scan(&rows).Error
	return rows, err
}

func (m *defaultCaddyLogModel) CountUniqueVisitor(ctx context.Context, start, end time.Time) (int64, error) {
	var total int64
	err := conn(m.db, ctx).Raw(
		`SELECT COUNT(DISTINCT COALESCE(NULLIF(client_ip, ''), remote_ip))
		 FROM caddy_logs
		 WHERE log_time BETWEEN ? AND ? AND (client_ip <> '' OR remote_ip <> '')`,
		start, end,
	).Scan(&total).Error
	return total, err
}

func (m *defaultCaddyLogModel) CountUniqueRemoteIP(ctx context.Context, start, end time.Time) (int64, error) {
	var total int64
	err := conn(m.db, ctx).Raw(
		`SELECT COUNT(DISTINCT COALESCE(NULLIF(client_ip, ''), remote_ip))
		 FROM caddy_logs
		 WHERE log_time BETWEEN ? AND ? AND (client_ip <> '' OR remote_ip <> '')`,
		start, end,
	).Scan(&total).Error
	return total, err
}

func (m *defaultCaddyLogModel) CountAttackIP(ctx context.Context, start, end time.Time, statuses []int) (int64, error) {
	var total int64
	err := conn(m.db, ctx).Raw(
		`SELECT COUNT(DISTINCT COALESCE(NULLIF(client_ip, ''), remote_ip))
		 FROM caddy_logs
		 WHERE log_time BETWEEN ? AND ? AND status IN ? AND (client_ip <> '' OR remote_ip <> '')`,
		start, end, statuses,
	).Scan(&total).Error
	return total, err
}

func (m *defaultCaddyLogModel) TrendRows(ctx context.Context, start, end time.Time, intervalSec int) ([]DashboardTrendRow, error) {
	rows := make([]DashboardTrendRow, 0)
	err := conn(m.db, ctx).Raw(
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
	err := conn(m.db, ctx).Raw(
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

func (m *defaultCaddyLogModel) ProvinceRows(ctx context.Context, start, end time.Time, limit int) ([]DashboardGeoRow, error) {
	rows := make([]DashboardGeoRow, 0)
	err := conn(m.db, ctx).Raw(
		`SELECT COALESCE(NULLIF(province, ''), '未知') AS name, COUNT(*) AS value
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
	err := conn(m.db, ctx).Model(&CaddyLog{}).
		Where("log_time >= ? AND log_time <= ?", start, end).
		Order("log_time desc, id desc").
		Limit(limit).
		Find(&logs).Error
	return logs, err
}

func (m *defaultCaddyLogModel) baseRange(ctx context.Context, start, end time.Time) *gorm.DB {
	return conn(m.db, ctx).Model(&CaddyLog{}).Where("log_time >= ? AND log_time <= ?", start, end)
}

func logOrder(sortBy, order string) string {
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
