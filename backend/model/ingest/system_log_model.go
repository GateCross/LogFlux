package ingest

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"
)

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

type SystemLogModel interface {
	List(ctx context.Context, query SystemLogQuery) ([]SystemLog, int64, error)
	Clear(ctx context.Context) error
}

type defaultSystemLogModel struct {
	db *gorm.DB
}

func NewSystemLogModel(db *gorm.DB) SystemLogModel {
	return &defaultSystemLogModel{db: db}
}

func systemLogConn(db *gorm.DB, ctx context.Context) *gorm.DB {
	if ctx == nil {
		ctx = context.Background()
	}
	return db.WithContext(ctx)
}

func (m *defaultSystemLogModel) List(ctx context.Context, query SystemLogQuery) ([]SystemLog, int64, error) {
	db := systemLogConn(m.db, ctx).Model(&SystemLog{})
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
	if err := db.Order(systemLogOrder(query.SortBy, query.Order)).Limit(pageSize).Offset((page - 1) * pageSize).Find(&logs).Error; err != nil {
		return nil, 0, err
	}
	return logs, total, nil
}

func (m *defaultSystemLogModel) Clear(ctx context.Context) error {
	return systemLogConn(m.db, ctx).Exec(`TRUNCATE TABLE "system_logs" RESTART IDENTITY`).Error
}

func systemLogOrder(sortBy, order string) string {
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
