package model

import "time"

// WAFSource WAF 更新源配置
type WAFSource struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	Name        string `gorm:"size:100;uniqueIndex;not null" json:"name"`
	Kind        string `gorm:"size:32;index;not null;default:'crs'" json:"kind"`    // crs | coraza_engine
	Mode        string `gorm:"size:32;index;not null;default:'remote'" json:"mode"` // remote | manual
	URL         string `gorm:"type:text" json:"url,omitempty"`
	ChecksumURL string `gorm:"type:text" json:"checksumUrl,omitempty"`

	AuthType   string `gorm:"size:20;not null;default:'none'" json:"authType"` // none | token | basic
	AuthSecret string `gorm:"type:text" json:"authSecret,omitempty"`

	Schedule string `gorm:"size:120" json:"schedule,omitempty"`

	Enabled      bool `gorm:"default:true;index;not null" json:"enabled"`
	AutoCheck    bool `gorm:"default:true;not null" json:"autoCheck"`
	AutoDownload bool `gorm:"default:true;not null" json:"autoDownload"`
	AutoActivate bool `gorm:"default:false;not null" json:"autoActivate"`

	LastCheckedAt *time.Time `json:"lastCheckedAt,omitempty"`
	LastRelease   string     `gorm:"size:120" json:"lastRelease,omitempty"`
	LastError     string     `gorm:"type:text" json:"lastError,omitempty"`
	Meta          JSONMap    `gorm:"type:jsonb" json:"meta,omitempty"`
}

func (WAFSource) TableName() string {
	return "waf_sources"
}
