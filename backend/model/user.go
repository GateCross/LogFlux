package model

import (
	"time"

	"github.com/lib/pq"
	"gorm.io/gorm"
)

type User struct {
	ID        uint `gorm:"primarykey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
	Username  string         `gorm:"uniqueIndex;size:255;not null"`
	Password  string         `gorm:"size:255;not null"`
	Roles     pq.StringArray `gorm:"type:text[];not null;default:'{}'"`
}

func (User) TableName() string {
	return "users"
}
