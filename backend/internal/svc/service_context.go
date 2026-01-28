package svc

import (
	"logflux/common/gorm"
	"logflux/internal/config"
	"logflux/internal/ingest"
	"logflux/model"

	"github.com/lib/pq"
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
	db.AutoMigrate(&model.User{}, &model.CaddyLog{}, &model.LogSource{}, &model.Role{}, &model.Menu{}, &model.CaddyServer{})

	// 初始化 RBAC 数据
	initRBACData(db)

	// Create default admin user if not exists
	var count int64
	db.Model(&model.User{}).Where("username = ?", "admin").Count(&count)
	if count == 0 {
		db.Create(&model.User{
			Username: "admin",
			Password: "123456", // In real app, use hash
			Roles:    []string{"admin"},
		})
	} else {
		// 确保 admin 用户拥有 admin 角色（修复旧数据问题）
		var user model.User
		db.Where("username = ?", "admin").First(&user)
		if len(user.Roles) == 0 {
			db.Model(&user).Update("roles", pq.StringArray{"admin"})
		}
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

// initRBACData 初始化 RBAC 角色和菜单数据
func initRBACData(db *gorm2.DB) {
	// 初始化默认角色
	roles := []model.Role{
		{
			Name:        "admin",
			DisplayName: "管理员",
			Description: "系统管理员，拥有所有权限",
			Permissions: []string{"dashboard", "manage", "manage_user", "manage_role", "logs", "logs_caddy"},
		},
		{
			Name:        "analyst",
			DisplayName: "分析师",
			Description: "数据分析师，可以查看和分析日志",
			Permissions: []string{"dashboard", "logs", "logs_caddy"},
		},
		{
			Name:        "viewer",
			DisplayName: "访客",
			Description: "只读访问权限",
			Permissions: []string{"dashboard"},
		},
	}

	for _, role := range roles {
		var count int64
		db.Model(&model.Role{}).Where("name = ?", role.Name).Count(&count)
		if count == 0 {
			db.Create(&role)
		} else {
			// 更新已存在角色的权限
			db.Model(&model.Role{}).Where("name = ?", role.Name).Updates(map[string]interface{}{
				"permissions": pq.StringArray(role.Permissions),
			})
		}
	}

	// 初始化菜单数据
	menus := []model.Menu{
		{
			Name:          "dashboard",
			Path:          "/dashboard",
			Component:     "layout.base",
			Order:         1,
			Meta:          `{"title":"dashboard","i18nKey":"route.dashboard","icon":"mdi:monitor-dashboard"}`,
			RequiredRoles: []string{"admin", "analyst", "viewer"},
		},
		{
			Name:          "logs",
			Path:          "/logs",
			Component:     "layout.base",
			Order:         5,
			Meta:          `{"title":"logs","i18nKey":"route.logs","icon":"mdi:file-document-multiple"}`,
			RequiredRoles: []string{"admin", "analyst"},
		},
		{
			Name:          "manage",
			Path:          "/manage",
			Component:     "layout.base",
			Order:         9,
			Meta:          `{"title":"manage","i18nKey":"route.manage","icon":"carbon:cloud-service-management"}`,
			RequiredRoles: []string{"admin"},
		},
	}

	for _, menu := range menus {
		var count int64
		db.Model(&model.Menu{}).Where("name = ?", menu.Name).Count(&count)
		if count == 0 {
			db.Create(&menu)
		}
	}
}
