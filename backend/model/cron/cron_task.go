package cron

import (
	"time"
)

type CronTask struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	Name          string `gorm:"type:varchar(100);not null;uniqueIndex" json:"name"`
	Schedule      string `gorm:"type:varchar(100);not null" json:"schedule"` // Cron expression
	ScriptMode    string `gorm:"type:varchar(16);not null;default:inline" json:"scriptMode"`
	Script        string `gorm:"type:text" json:"script"`              // Shell script to execute when using inline mode
	CurrentFileID uint   `gorm:"default:0;index" json:"currentFileId"` // Active uploaded script file
	Status        int    `gorm:"default:1;not null" json:"status"`     // 1: Enabled, 0: Disabled
	Timeout       int    `gorm:"default:60" json:"timeout"`            // Execution timeout in seconds
}

func (CronTask) TableName() string {
	return "cron_tasks"
}
