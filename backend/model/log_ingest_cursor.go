package model

import "time"

type LogIngestCursor struct {
	ID        uint `gorm:"primarykey"`
	CreatedAt time.Time
	UpdatedAt time.Time

	FilePath string `gorm:"size:1024;not null;uniqueIndex"`
	Offset   int64  `gorm:"not null;default:0"`
}

func (LogIngestCursor) TableName() string {
	return "log_ingest_cursors"
}
