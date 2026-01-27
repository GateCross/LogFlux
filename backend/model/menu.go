package model

import (
	"time"

	"github.com/lib/pq"
)

// Menu 菜单模型
type Menu struct {
	ID        uint `gorm:"primarykey"`
	CreatedAt time.Time
	UpdatedAt time.Time

	Name      string `gorm:"uniqueIndex;size:100;not null;comment:菜单唯一标识"`
	Path      string `gorm:"size:255;not null;comment:菜单路径"`
	Component string `gorm:"size:255;comment:组件路径"`
	ParentID  *uint  `gorm:"index;comment:父菜单ID"`
	Order     int    `gorm:"default:0;comment:排序"`

	// JSONB 字段，存储 meta 信息（title, icon, i18nKey 等）
	Meta string `gorm:"type:jsonb;comment:菜单元数据"`

	// 需要的角色列表
	RequiredRoles pq.StringArray `gorm:"type:text[];not null;default:'{}';comment:所需角色"`

	// 关联关系
	Parent   *Menu  `gorm:"foreignKey:ParentID"`
	Children []Menu `gorm:"foreignKey:ParentID"`
}

func (Menu) TableName() string {
	return "menus"
}
