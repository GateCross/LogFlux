package model

import (
	"time"

	"gorm.io/gorm"
)

type CaddyLog struct {
	ID        uint      `gorm:"primarykey"`
	CreatedAt time.Time `gorm:"index"` // Index for time-range queries
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`

	// Parsed from [{ts}]
	LogTime time.Time `gorm:"index;not null"`

	// GeoIP fields
	Country  string `gorm:"size:100;index"`
	Province string `gorm:"size:100"`
	City     string `gorm:"size:100"`

	// Request fields
	Host   string `gorm:"size:255;index"`
	Method string `gorm:"size:10"`
	Uri    string `gorm:"type:text"` // URLs can be long
	Proto  string `gorm:"size:20"`
	Status int    `gorm:"index"`
	Size   int64

	// Client info
	UserAgent string `gorm:"type:text"`
	RemoteIP  string `gorm:"size:50;index"`
	ClientIP  string `gorm:"size:50"`
}

func (CaddyLog) TableName() string {
	return "caddy_logs"
}
