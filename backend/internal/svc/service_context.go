package svc

import (
	"logflux/common/gorm"
	"logflux/internal/config"
	"logflux/internal/ingest"
	"logflux/model"

	gorm2 "gorm.io/gorm"
)

type ServiceContext struct {
	Config   config.Config
	DB       *gorm2.DB
	Ingestor *ingest.CaddyIngestor
}

func NewServiceContext(c config.Config) *ServiceContext {
	db := gorm.InitGorm(c.Database.DSN())
	// Auto Migrate
	db.AutoMigrate(&model.User{}, &model.CaddyLog{}, &model.LogSource{})

	// Create default admin user if not exists
	var count int64
	db.Model(&model.User{}).Where("username = ?", "admin").Count(&count)
	if count == 0 {
		db.Create(&model.User{
			Username: "admin",
			Password: "123456", // In real app, use hash
			Roles:    []string{"admin"},
		})
	}

	// Init Ingestor
	ingestor := ingest.NewCaddyIngestor(db)

	// Load enabled sources from DB
	var sources []model.LogSource
	db.Where("enabled = ?", true).Find(&sources)
	for _, source := range sources {
		ingestor.Start(source.Path)
	}

	// Legacy config support (migration)
	if c.CaddyLogPath != "" {
		// Check if exists in DB, if not add it
		var cnt int64
		db.Model(&model.LogSource{}).Where("path = ?", c.CaddyLogPath).Count(&cnt)
		if cnt == 0 {
			db.Create(&model.LogSource{
				Name:    "Default Config",
				Path:    c.CaddyLogPath,
				Type:    "caddy",
				Enabled: true,
			})
			ingestor.Start(c.CaddyLogPath)
		}
	}

	return &ServiceContext{
		Config:   c,
		DB:       db,
		Ingestor: ingestor,
	}
}
