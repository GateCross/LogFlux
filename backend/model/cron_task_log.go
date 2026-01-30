package model

import (
	"time"
)

type CronTaskLog struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"createdAt"`

	TaskID    uint      `gorm:"not null;index" json:"taskId"`
	StartTime time.Time `gorm:"not null" json:"startTime"`
	EndTime   time.Time `json:"endTime"`
	Status    int       `gorm:"not null" json:"status"` // 0: Running, 1: Success, 2: Failed, 3: Timeout
	ExitCode  int       `json:"exitCode"`
	Output    string    `gorm:"type:text" json:"output"`
	Error     string    `gorm:"type:text" json:"error"`
	Duration  int64     `json:"duration"` // in milliseconds

	// Relationships
	Task CronTask `gorm:"foreignKey:TaskID" json:"task"`
}

func (CronTaskLog) TableName() string {
	return "cron_task_logs"
}
