package model

import "time"

// WafPolicyFalsePositiveFeedback 人工标注的误报反馈记录
type WafPolicyFalsePositiveFeedback struct {
	ID        uint      `gorm:"primarykey"`
	CreatedAt time.Time `gorm:"index"`
	UpdatedAt time.Time

	PolicyID       uint       `gorm:"index;not null;default:0"` // 0 代表全局/未指定策略
	Host           string     `gorm:"size:255;index"`
	Path           string     `gorm:"size:512;index"`
	Method         string     `gorm:"size:16;index"`
	Status         int        `gorm:"index"`
	FeedbackStatus string     `gorm:"size:32;not null;default:pending;index"`
	Assignee       string     `gorm:"size:64;index"`
	DueAt          *time.Time `gorm:"index"`
	SampleURI      string     `gorm:"type:text"`
	Reason         string     `gorm:"type:text;not null"`
	Suggestion     string     `gorm:"type:text"`
	Operator       string     `gorm:"size:64;index"`
	ProcessNote    string     `gorm:"type:text"`
	ProcessedBy    string     `gorm:"size:64;index"`
	ProcessedAt    *time.Time `gorm:"index"`
}

func (WafPolicyFalsePositiveFeedback) TableName() string {
	return "waf_policy_false_positive_feedbacks"
}
