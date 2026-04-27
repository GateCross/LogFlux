package model

import (
	"context"

	"gorm.io/gorm"
)

// MenuModel 定义菜单表的数据访问能力。
type MenuModel interface {
	FindAll(ctx context.Context) ([]Menu, error)
	FindByID(ctx context.Context, id uint) (*Menu, error)
	CountByName(ctx context.Context, name string) (int64, error)
	CountChildren(ctx context.Context, parentID uint) (int64, error)
	Create(ctx context.Context, menu *Menu) error
	UpdateFields(ctx context.Context, menu *Menu, updates map[string]interface{}) error
	DeleteByID(ctx context.Context, id uint) error
}

type defaultMenuModel struct {
	db *gorm.DB
}

// NewMenuModel 创建菜单模型。
func NewMenuModel(db *gorm.DB) MenuModel {
	return &defaultMenuModel{db: db}
}

func (m *defaultMenuModel) conn(ctx context.Context) *gorm.DB {
	if ctx == nil {
		ctx = context.Background()
	}
	return m.db.WithContext(ctx)
}

func (m *defaultMenuModel) FindAll(ctx context.Context) ([]Menu, error) {
	var menus []Menu
	if err := m.conn(ctx).Order("\"order\" asc, id asc").Find(&menus).Error; err != nil {
		return nil, err
	}
	return menus, nil
}

func (m *defaultMenuModel) FindByID(ctx context.Context, id uint) (*Menu, error) {
	var menu Menu
	if err := m.conn(ctx).First(&menu, id).Error; err != nil {
		return nil, err
	}
	return &menu, nil
}

func (m *defaultMenuModel) CountByName(ctx context.Context, name string) (int64, error) {
	var count int64
	if err := m.conn(ctx).Model(&Menu{}).Where("name = ?", name).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (m *defaultMenuModel) CountChildren(ctx context.Context, parentID uint) (int64, error) {
	var count int64
	if err := m.conn(ctx).Model(&Menu{}).Where("parent_id = ?", parentID).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (m *defaultMenuModel) Create(ctx context.Context, menu *Menu) error {
	return m.conn(ctx).Create(menu).Error
}

func (m *defaultMenuModel) UpdateFields(ctx context.Context, menu *Menu, updates map[string]interface{}) error {
	return m.conn(ctx).Model(menu).Updates(updates).Error
}

func (m *defaultMenuModel) DeleteByID(ctx context.Context, id uint) error {
	return m.conn(ctx).Delete(&Menu{}, id).Error
}
