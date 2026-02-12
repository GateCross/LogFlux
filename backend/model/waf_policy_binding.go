package model

import "time"

// WafPolicyBinding 策略作用域绑定
type WafPolicyBinding struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	PolicyID uint `gorm:"index;not null" json:"policyId"`
	Policy   WafPolicy

	Name        string `gorm:"size:120;not null;default:''" json:"name"`
	Description string `gorm:"size:255" json:"description,omitempty"`
	Enabled     bool   `gorm:"index;not null;default:true" json:"enabled"`

	ScopeType string `gorm:"size:16;index;not null;default:'global'" json:"scopeType"` // global | site | route
	Host      string `gorm:"size:255;index;not null;default:''" json:"host"`
	Path      string `gorm:"size:255;index;not null;default:''" json:"path"`
	Method    string `gorm:"size:32;index;not null;default:''" json:"method"`
	Priority  int64  `gorm:"index;not null;default:100" json:"priority"`
}

func (WafPolicyBinding) TableName() string {
	return "waf_policy_bindings"
}
