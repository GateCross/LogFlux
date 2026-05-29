package system

import (
	"time"

	"gorm.io/gorm"
)

type SystemConfig struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	Key       string    `gorm:"size:100;uniqueIndex;not null" json:"key"`
	Value     string    `gorm:"type:text;not null" json:"value"` // JSON string
}

func (SystemConfig) TableName() string {
	return "system_configs"
}

type SystemConfigModel struct {
	db *gorm.DB
}

func NewSystemConfigModel(db *gorm.DB) *SystemConfigModel {
	return &SystemConfigModel{db: db}
}

func (m *SystemConfigModel) GetByKey(key string) (*SystemConfig, error) {
	var cfg SystemConfig
	err := m.db.Where("key = ?", key).First(&cfg).Error
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (m *SystemConfigModel) SetByKey(key, value string) error {
	return m.db.Where(SystemConfig{Key: key}).
		Assign(SystemConfig{Value: value}).
		FirstOrCreate(&SystemConfig{}).Error
}
