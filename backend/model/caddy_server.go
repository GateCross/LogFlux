package model

import (
	"time"
)

type CaddyServer struct {
	ID        uint `gorm:"primarykey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	Name      string `gorm:"uniqueIndex;not null"`
	Url       string `gorm:"not null"`
	Token     string
	Type      string `gorm:"default:'local'"` // local or remote
	Username  string
	Password  string
	Config    string `gorm:"type:text"` // Store Caddyfile content
}

func (CaddyServer) TableName() string {
	return "caddy_servers"
}
