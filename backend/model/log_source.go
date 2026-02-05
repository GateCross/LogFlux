package model

import (
	"time"
)

type LogSource struct {
	ID        uint `gorm:"primarykey"`
	CreatedAt time.Time
	UpdatedAt time.Time

	Name         string `gorm:"size:255;not null"`
	Path         string `gorm:"size:1024;not null;uniqueIndex"` // File path
	Type         string `gorm:"size:50;default:'caddy'"`        // Source type (caddy, nginx, etc)
	Enabled      bool   `gorm:"default:true"`                   // Is monitoring active?
	ScanInterval int    `gorm:"default:60"`                     // Directory scan interval (seconds)
}

func (LogSource) TableName() string {
	return "log_sources"
}
