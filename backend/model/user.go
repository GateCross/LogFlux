package model

import (
	"time"

	"github.com/lib/pq"
)

type User struct {
	ID        uint `gorm:"primarykey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	Username  string         `gorm:"uniqueIndex;size:255;not null"`
	Password  string         `gorm:"size:255;not null"`
	Roles     pq.StringArray `gorm:"type:text[];not null;default:'{}'"`
	Status    int            `gorm:"default:1;not null"` // 1=启用, 0=禁用
}

func (User) TableName() string {
	return "users"
}
