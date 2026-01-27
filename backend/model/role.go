package model

import (
	"time"

	"github.com/lib/pq"
)

// Role 角色模型
type Role struct {
	ID        uint `gorm:"primarykey"`
	CreatedAt time.Time
	UpdatedAt time.Time

	Name        string `gorm:"uniqueIndex;size:50;not null;comment:角色名称"`
	DisplayName string `gorm:"size:100;not null;comment:显示名称"`
	Description string `gorm:"type:text;comment:角色描述"`

	// 权限列表，存储菜单路由名称
	Permissions pq.StringArray `gorm:"type:text[];not null;default:'{}';comment:权限列表"`
}

func (Role) TableName() string {
	return "roles"
}
