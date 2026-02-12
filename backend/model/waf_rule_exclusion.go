package model

import "time"

// WafRuleExclusion 误报治理例外规则
type WafRuleExclusion struct {
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

	RemoveType  string `gorm:"size:16;not null;default:'id'" json:"removeType"` // id | tag
	RemoveValue string `gorm:"size:255;not null;default:''" json:"removeValue"`
}

func (WafRuleExclusion) TableName() string {
	return "waf_rule_exclusions"
}
