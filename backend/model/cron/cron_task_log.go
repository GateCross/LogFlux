package cron

import (
	"time"
)

type CronTaskLog struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"createdAt"`

	TaskID    uint      `gorm:"not null;index" json:"taskId"`
	TaskName  string    `gorm:"type:varchar(100);not null;default:''" json:"taskName"`
	StartTime time.Time `gorm:"not null" json:"startTime"`
	EndTime   time.Time `json:"endTime"`
	Status    int       `gorm:"not null" json:"status"` // 0: Running, 1: Success, 2: Failed, 3: Timeout
	ExitCode  int       `json:"exitCode"`
	Output    string    `gorm:"type:text" json:"output"` // Standard output
	Error     string    `gorm:"type:text" json:"error"`  // Standard error or execution message
	Duration  int64     `json:"duration"`                // in milliseconds

	TriggerMode       string `gorm:"type:varchar(32);not null;default:manual" json:"triggerMode"`
	ScriptMode        string `gorm:"type:varchar(16);not null;default:inline" json:"scriptMode"`
	ScriptSnapshot    string `gorm:"type:text" json:"scriptSnapshot"`
	ScriptFileID      uint   `gorm:"default:0;index" json:"scriptFileId"`
	ScriptFileVersion int    `gorm:"default:0" json:"scriptFileVersion"`
	ScriptFileName    string `gorm:"type:varchar(255)" json:"scriptFileName"`
	ScriptFilePath    string `gorm:"type:text" json:"scriptFilePath"`
	ScriptFileSHA256  string `gorm:"type:varchar(64)" json:"scriptFileSha256"`
}

func (CronTaskLog) TableName() string {
	return "cron_task_logs"
}
