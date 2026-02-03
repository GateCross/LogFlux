package model

import "time"

type CaddyConfigHistory struct {
	ID        uint      `gorm:"primarykey"`
	CreatedAt time.Time `gorm:"index"`
	UpdatedAt time.Time
	ServerID  uint   `gorm:"index;not null"`
	Action    string `gorm:"size:20;default:'update'"`
	Hash      string `gorm:"size:64;index"`
	Config    string `gorm:"type:text"`
	Modules   string `gorm:"type:jsonb"`
}

func (CaddyConfigHistory) TableName() string {
	return "caddy_config_history"
}
