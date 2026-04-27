package model

import (
	"context"

	"gorm.io/gorm"
)

type LogSourceModel interface {
	Create(ctx context.Context, source *LogSource) error
	FindByID(ctx context.Context, id uint) (*LogSource, error)
	List(ctx context.Context, page, pageSize int) ([]LogSource, int64, error)
	Save(ctx context.Context, source *LogSource) error
	DeleteByID(ctx context.Context, id uint) error
}

type defaultLogSourceModel struct {
	db *gorm.DB
}

func NewLogSourceModel(db *gorm.DB) LogSourceModel {
	return &defaultLogSourceModel{db: db}
}

func (m *defaultLogSourceModel) conn(ctx context.Context) *gorm.DB {
	if ctx == nil {
		ctx = context.Background()
	}
	return m.db.WithContext(ctx)
}

func (m *defaultLogSourceModel) Create(ctx context.Context, source *LogSource) error {
	return m.conn(ctx).Create(source).Error
}

func (m *defaultLogSourceModel) FindByID(ctx context.Context, id uint) (*LogSource, error) {
	var source LogSource
	if err := m.conn(ctx).First(&source, id).Error; err != nil {
		return nil, err
	}
	return &source, nil
}

func (m *defaultLogSourceModel) List(ctx context.Context, page, pageSize int) ([]LogSource, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	db := m.conn(ctx).Model(&LogSource{})

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var sources []LogSource
	offset := (page - 1) * pageSize
	if err := db.Order("created_at desc, id desc").Limit(pageSize).Offset(offset).Find(&sources).Error; err != nil {
		return nil, 0, err
	}
	return sources, total, nil
}

func (m *defaultLogSourceModel) Save(ctx context.Context, source *LogSource) error {
	return m.conn(ctx).Save(source).Error
}

func (m *defaultLogSourceModel) DeleteByID(ctx context.Context, id uint) error {
	return m.conn(ctx).Delete(&LogSource{}, id).Error
}
