package model

import "time"

// WafPolicyRevision WAF 策略发布快照
type WafPolicyRevision struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	PolicyID uint `gorm:"index;not null" json:"policyId"`
	Policy   WafPolicy

	Version uint   `gorm:"not null;default:1" json:"version"`
	Status  string `gorm:"size:32;index;not null;default:'draft'" json:"status"` // draft | published | rolled_back

	ConfigSnapshot     JSONMap `gorm:"type:jsonb" json:"configSnapshot,omitempty"`
	DirectivesSnapshot string  `gorm:"type:text" json:"directivesSnapshot,omitempty"`

	Operator string `gorm:"size:100" json:"operator,omitempty"`
	Message  string `gorm:"type:text" json:"message,omitempty"`
}

func (WafPolicyRevision) TableName() string {
	return "waf_policy_revisions"
}

