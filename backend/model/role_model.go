package model

import (
	"context"

	"github.com/lib/pq"
	"gorm.io/gorm"
)

// RoleModel 定义角色表的数据访问能力。
type RoleModel interface {
	FindAll(ctx context.Context) ([]Role, error)
	FindByID(ctx context.Context, id uint) (*Role, error)
	FindByNames(ctx context.Context, names []string) ([]Role, error)
	UpdatePermissions(ctx context.Context, role *Role, permissions interface{}) error
}

type defaultRoleModel struct {
	db *gorm.DB
}

// NewRoleModel 创建角色模型。
func NewRoleModel(db *gorm.DB) RoleModel {
	return &defaultRoleModel{db: db}
}

func (m *defaultRoleModel) conn(ctx context.Context) *gorm.DB {
	if ctx == nil {
		ctx = context.Background()
	}
	return m.db.WithContext(ctx)
}

func (m *defaultRoleModel) FindAll(ctx context.Context) ([]Role, error) {
	var roles []Role
	if err := m.conn(ctx).Order("id asc").Find(&roles).Error; err != nil {
		return nil, err
	}
	return roles, nil
}

func (m *defaultRoleModel) FindByID(ctx context.Context, id uint) (*Role, error) {
	var role Role
	if err := m.conn(ctx).First(&role, id).Error; err != nil {
		return nil, err
	}
	return &role, nil
}

func (m *defaultRoleModel) FindByNames(ctx context.Context, names []string) ([]Role, error) {
	if len(names) == 0 {
		return []Role{}, nil
	}
	var roles []Role
	if err := m.conn(ctx).Where("name = ANY(?)", pq.Array(names)).Find(&roles).Error; err != nil {
		return nil, err
	}
	return roles, nil
}

func (m *defaultRoleModel) UpdatePermissions(ctx context.Context, role *Role, permissions interface{}) error {
	return m.conn(ctx).Model(role).Update("permissions", permissions).Error
}
