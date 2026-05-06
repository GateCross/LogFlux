package cron

import "time"

// CronTaskFile 保存定时任务脚本文件的历史版本。
type CronTaskFile struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	TaskID       uint   `gorm:"not null;index:idx_cron_task_files_task_version,unique" json:"taskId"`
	Version      int    `gorm:"not null;index:idx_cron_task_files_task_version,unique" json:"version"`
	OriginalName string `gorm:"type:varchar(255);not null" json:"originalName"`
	StoredName   string `gorm:"type:varchar(255);not null" json:"storedName"`
	FilePath     string `gorm:"type:text;not null" json:"filePath"`
	SizeBytes    int64  `gorm:"not null" json:"sizeBytes"`
	SHA256       string `gorm:"type:varchar(64);not null" json:"sha256"`
	IsCurrent    bool   `gorm:"default:false;not null;index" json:"isCurrent"`
}

func (CronTaskFile) TableName() string {
	return "cron_task_files"
}
