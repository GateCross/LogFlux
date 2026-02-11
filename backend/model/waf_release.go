package model

import "time"

// WafRelease WAF 规则发布版本
type WafRelease struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	SourceID uint `gorm:"index:idx_waf_release_source_version,priority:1;index" json:"sourceId"`
	Source   WafSource

	Kind         string `gorm:"size:32;index;not null;default:'crs'" json:"kind"` // crs | coraza_engine
	Version      string `gorm:"size:120;not null;index:idx_waf_release_source_version,priority:2" json:"version"`
	ArtifactType string `gorm:"size:32;not null;default:'tar.gz'" json:"artifactType"` // tar.gz | zip | upload

	Checksum    string `gorm:"size:128" json:"checksum,omitempty"`
	SizeBytes   int64  `gorm:"default:0" json:"sizeBytes"`
	StoragePath string `gorm:"type:text;not null" json:"storagePath"`

	Status string  `gorm:"size:32;index;not null;default:'downloaded'" json:"status"` // downloaded | verified | active | failed | rolled_back
	Meta   JSONMap `gorm:"type:jsonb" json:"meta,omitempty"`
}

func (WafRelease) TableName() string {
	return "waf_releases"
}
