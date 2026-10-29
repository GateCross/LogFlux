package caddy

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// 站点图标持久化行，记录域名对应的 data URL 图标。
type SiteIcon struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	Host      string    `gorm:"size:255;not null;uniqueIndex:uk_caddy_site_icons_host" json:"host"`
	Icon      string    `gorm:"type:text;not null" json:"icon"`
}

func (SiteIcon) TableName() string {
	return "caddy_site_icons"
}

type SiteIconModel interface {
	FindByHost(ctx context.Context, host string) (*SiteIcon, error)
	Upsert(ctx context.Context, host, icon string) error
}

type defaultSiteIconModel struct {
	db *gorm.DB
}

func NewSiteIconModel(db *gorm.DB) SiteIconModel {
	return &defaultSiteIconModel{db: db}
}

func (m *defaultSiteIconModel) conn(ctx context.Context) *gorm.DB {
	if ctx == nil {
		ctx = context.Background()
	}
	return m.db.WithContext(ctx)
}

func (m *defaultSiteIconModel) FindByHost(ctx context.Context, host string) (*SiteIcon, error) {
	var row SiteIcon
	if err := m.conn(ctx).Where("host = ?", host).First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (m *defaultSiteIconModel) Upsert(ctx context.Context, host, icon string) error {
	row := SiteIcon{
		Host:      host,
		Icon:      icon,
		UpdatedAt: time.Now(),
	}
	return m.conn(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "host"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"icon", "updated_at",
		}),
	}).Create(&row).Error
}
