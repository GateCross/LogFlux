package model

import (
	"context"
	"strings"

	"gorm.io/gorm"
)

type CronTaskModel interface {
	Create(ctx context.Context, task *CronTask) error
	DeleteByID(ctx context.Context, id uint) error
	FindByID(ctx context.Context, id uint) (*CronTask, error)
	UpdateFields(ctx context.Context, task *CronTask, updates map[string]interface{}) error
	List(ctx context.Context, name string, page, pageSize int) ([]CronTask, int64, error)
	ListLogs(ctx context.Context, taskID uint, status int, page, pageSize int) ([]CronTaskLog, int64, error)
}

type defaultCronTaskModel struct {
	db *gorm.DB
}

func NewCronTaskModel(db *gorm.DB) CronTaskModel {
	return &defaultCronTaskModel{db: db}
}

func (m *defaultCronTaskModel) conn(ctx context.Context) *gorm.DB {
	if ctx == nil {
		ctx = context.Background()
	}
	return m.db.WithContext(ctx)
}

func (m *defaultCronTaskModel) Create(ctx context.Context, task *CronTask) error {
	return m.conn(ctx).Create(task).Error
}

func (m *defaultCronTaskModel) DeleteByID(ctx context.Context, id uint) error {
	return m.conn(ctx).Delete(&CronTask{}, id).Error
}

func (m *defaultCronTaskModel) FindByID(ctx context.Context, id uint) (*CronTask, error) {
	var task CronTask
	if err := m.conn(ctx).First(&task, id).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

func (m *defaultCronTaskModel) UpdateFields(ctx context.Context, task *CronTask, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}
	return m.conn(ctx).Model(task).Updates(updates).Error
}

func (m *defaultCronTaskModel) List(ctx context.Context, name string, page, pageSize int) ([]CronTask, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	db := m.conn(ctx).Model(&CronTask{})
	if keyword := strings.TrimSpace(name); keyword != "" {
		db = db.Where("name LIKE ?", "%"+keyword+"%")
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var tasks []CronTask
	offset := (page - 1) * pageSize
	if err := db.Offset(offset).Limit(pageSize).Order("id desc").Find(&tasks).Error; err != nil {
		return nil, 0, err
	}
	return tasks, total, nil
}

func (m *defaultCronTaskModel) ListLogs(ctx context.Context, taskID uint, status int, page, pageSize int) ([]CronTaskLog, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	db := m.conn(ctx).Model(&CronTaskLog{})
	if taskID > 0 {
		db = db.Where("task_id = ?", taskID)
	}
	if status >= 0 {
		db = db.Where("status = ?", status)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var logs []CronTaskLog
	offset := (page - 1) * pageSize
	if err := db.Preload("Task").Offset(offset).Limit(pageSize).Order("id desc").Find(&logs).Error; err != nil {
		return nil, 0, err
	}
	return logs, total, nil
}
