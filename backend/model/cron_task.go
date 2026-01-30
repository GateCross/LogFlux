package model

import (
	"time"
)

type CronTask struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	Name     string `gorm:"type:varchar(100);not null;uniqueIndex" json:"name"`
	Schedule string `gorm:"type:varchar(100);not null" json:"schedule"` // Cron expression
	Script   string `gorm:"type:text" json:"script"`                    // Shell script to execute
	Status   int    `gorm:"default:1;not null" json:"status"`           // 1: Enabled, 0: Disabled
	Timeout  int    `gorm:"default:60" json:"timeout"`                  // Execution timeout in seconds

	// Relationships
	Logs []CronTaskLog `gorm:"foreignKey:TaskID;constraint:OnDelete:CASCADE;" json:"-"`
}

func (CronTask) TableName() string {
	return "cron_tasks"
}
