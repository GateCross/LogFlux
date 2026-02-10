package model

import "time"

// WAFUpdateJob WAF 更新任务审计
type WAFUpdateJob struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	SourceID  uint `gorm:"index" json:"sourceId"`
	ReleaseID uint `gorm:"index" json:"releaseId"`

	Action      string `gorm:"size:32;index;not null" json:"action"`                       // check | download | verify | activate | rollback
	TriggerMode string `gorm:"size:32;index;not null;default:'manual'" json:"triggerMode"` // manual | schedule | upload
	Operator    string `gorm:"size:100" json:"operator,omitempty"`

	Status     string     `gorm:"size:20;index;not null;default:'running'" json:"status"` // running | success | failed
	Message    string     `gorm:"type:text" json:"message,omitempty"`
	StartedAt  *time.Time `json:"startedAt,omitempty"`
	FinishedAt *time.Time `json:"finishedAt,omitempty"`
	Meta       JSONMap    `gorm:"type:jsonb" json:"meta,omitempty"`
}

func (WAFUpdateJob) TableName() string {
	return "waf_update_jobs"
}
