package model

import "time"

// NotificationJob 异步派发任务表（队列真源）
type NotificationJob struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	LogID uint `gorm:"index;not null" json:"log_id"`

	ChannelID     uint   `gorm:"index;not null" json:"channel_id"`
	ProviderType  string `gorm:"size:50;index;not null" json:"provider_type"`
	EventType     string `gorm:"size:100;index;not null" json:"event_type"`
	EventLevel    string `gorm:"size:20;index" json:"event_level"`
	EventTitle    string `gorm:"type:text" json:"event_title"`
	EventMessage  string `gorm:"type:text" json:"event_message"`
	EventData     JSONMap `gorm:"type:jsonb" json:"event_data,omitempty"`
	TemplateName  string `gorm:"type:text" json:"template_name,omitempty"`

	Status      string     `gorm:"size:50;index;not null" json:"status"` // queued, processing, succeeded, failed
	RetryCount  int        `gorm:"default:0" json:"retry_count"`
	NextRunAt   time.Time  `gorm:"index" json:"next_run_at"`
	LastError   string     `gorm:"type:text" json:"last_error,omitempty"`
	LastAttemptAt *time.Time `json:"last_attempt_at,omitempty"`
}

func (NotificationJob) TableName() string {
	return "notification_jobs"
}

const (
	NotificationJobStatusQueued      = "queued"
	NotificationJobStatusProcessing  = "processing"
	NotificationJobStatusSucceeded   = "succeeded"
	NotificationJobStatusFailed      = "failed"
)
