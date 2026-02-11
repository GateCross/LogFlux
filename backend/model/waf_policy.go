package model

import "time"

// WafPolicy WAF 运行时策略（结构化配置）
type WafPolicy struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	Name        string `gorm:"size:120;uniqueIndex;not null" json:"name"`
	Description string `gorm:"size:255" json:"description,omitempty"`

	Enabled   bool `gorm:"default:true;index;not null" json:"enabled"`
	IsDefault bool `gorm:"default:false;index;not null" json:"isDefault"`

	EngineMode            string `gorm:"size:32;not null;default:'on'" json:"engineMode"`                            // on | off | detectiononly
	AuditEngine           string `gorm:"size:32;not null;default:'relevantonly'" json:"auditEngine"`               // off | on | relevantonly
	AuditLogFormat        string `gorm:"size:32;not null;default:'json'" json:"auditLogFormat"`                    // json | native
	AuditRelevantStatus   string `gorm:"size:255;not null;default:'^(?:5|4(?!04))'" json:"auditRelevantStatus"`   // SecAuditLogRelevantStatus
	RequestBodyAccess     bool   `gorm:"default:true;not null" json:"requestBodyAccess"`                           // SecRequestBodyAccess
	RequestBodyLimit      int64  `gorm:"default:10485760;not null" json:"requestBodyLimit"`                        // SecRequestBodyLimit
	RequestBodyNoFilesLimit int64 `gorm:"default:1048576;not null" json:"requestBodyNoFilesLimit"`                // SecRequestBodyNoFilesLimit

	Config JSONMap `gorm:"type:jsonb" json:"config,omitempty"`
}

func (WafPolicy) TableName() string {
	return "waf_policies"
}

