package cron

import (
	"context"
	"strings"

	"gorm.io/gorm"
)

type CronTaskFileModel interface {
	Create(ctx context.Context, file *CronTaskFile) error
	DeleteByID(ctx context.Context, id uint) error
	DeleteByTaskID(ctx context.Context, taskID uint) error
	FindByID(ctx context.Context, id uint) (*CronTaskFile, error)
	FindCurrentByTaskID(ctx context.Context, taskID uint) (*CronTaskFile, error)
	FindLatestByTaskID(ctx context.Context, taskID uint) (*CronTaskFile, error)
	ListByTaskID(ctx context.Context, taskID uint, page, pageSize int) ([]CronTaskFile, int64, error)
	ActivateByID(ctx context.Context, taskID, fileID uint) error
}

type defaultCronTaskFileModel struct {
	db *gorm.DB
}

func NewCronTaskFileModel(db *gorm.DB) CronTaskFileModel {
	return &defaultCronTaskFileModel{db: db}
}

func (m *defaultCronTaskFileModel) conn(ctx context.Context) *gorm.DB {
	if ctx == nil {
		ctx = context.Background()
	}
	return m.db.WithContext(ctx)
}

func (m *defaultCronTaskFileModel) Create(ctx context.Context, file *CronTaskFile) error {
	return m.conn(ctx).Create(file).Error
}

func (m *defaultCronTaskFileModel) DeleteByID(ctx context.Context, id uint) error {
	return m.conn(ctx).Delete(&CronTaskFile{}, id).Error
}

func (m *defaultCronTaskFileModel) DeleteByTaskID(ctx context.Context, taskID uint) error {
	return m.conn(ctx).Where("task_id = ?", taskID).Delete(&CronTaskFile{}).Error
}

func (m *defaultCronTaskFileModel) FindByID(ctx context.Context, id uint) (*CronTaskFile, error) {
	var file CronTaskFile
	if err := m.conn(ctx).First(&file, id).Error; err != nil {
		return nil, err
	}
	return &file, nil
}

func (m *defaultCronTaskFileModel) FindCurrentByTaskID(ctx context.Context, taskID uint) (*CronTaskFile, error) {
	var file CronTaskFile
	if err := m.conn(ctx).Where("task_id = ? AND is_current = ?", taskID, true).First(&file).Error; err != nil {
		return nil, err
	}
	return &file, nil
}

func (m *defaultCronTaskFileModel) FindLatestByTaskID(ctx context.Context, taskID uint) (*CronTaskFile, error) {
	var file CronTaskFile
	if err := m.conn(ctx).Where("task_id = ?", taskID).Order("version desc").First(&file).Error; err != nil {
		return nil, err
	}
	return &file, nil
}

func (m *defaultCronTaskFileModel) ListByTaskID(ctx context.Context, taskID uint, page, pageSize int) ([]CronTaskFile, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	db := m.conn(ctx).Model(&CronTaskFile{}).Where("task_id = ?", taskID)
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var files []CronTaskFile
	offset := (page - 1) * pageSize
	if err := db.Offset(offset).Limit(pageSize).Order("version desc, id desc").Find(&files).Error; err != nil {
		return nil, 0, err
	}
	return files, total, nil
}

func (m *defaultCronTaskFileModel) ActivateByID(ctx context.Context, taskID, fileID uint) error {
	conn := m.conn(ctx)
	if err := conn.Model(&CronTaskFile{}).Where("task_id = ?", taskID).Update("is_current", false).Error; err != nil {
		return err
	}
	if err := conn.Model(&CronTaskFile{}).Where("id = ? AND task_id = ?", fileID, taskID).Update("is_current", true).Error; err != nil {
		return err
	}
	return nil
}

func sanitizeCronTaskFileName(name string) string {
	base := strings.TrimSpace(name)
	base = strings.ReplaceAll(base, "/", "_")
	base = strings.ReplaceAll(base, "\\", "_")
	base = strings.ReplaceAll(base, "..", "_")
	if base == "" {
		return "script.sh"
	}
	return base
}
