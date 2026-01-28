package svc

import (
	"context"
	"logflux/common/gorm"
	redisClient "logflux/common/redis"
	"logflux/internal/config"
	"logflux/internal/ingest"
	"logflux/internal/notification"
	"logflux/internal/notification/providers"
	"logflux/internal/tasks"
	"logflux/model"

	"github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	gorm2 "gorm.io/gorm"
)

type ServiceContext struct {
	Config          config.Config
	DB              *gorm2.DB
	Redis           *redis.Client
	Ingestor        *ingest.CaddyIngestor
	ArchiveTask     *tasks.ArchiveTask
	NotificationMgr notification.NotificationManager
}

func NewServiceContext(c config.Config) *ServiceContext {
	db := gorm.InitGorm(c.Database.DSN())

	// Auto Migrate - 包含归档表和通知表
	db.AutoMigrate(
		&model.User{},
		&model.CaddyLog{},
		&model.CaddyLogArchive{}, // 归档表
		&model.LogSource{},
		&model.Role{},
		&model.Menu{},
		&model.CaddyServer{},
		// 通知相关表
		&model.NotificationChannel{},
		&model.NotificationRule{},
		&model.NotificationLog{},
	)

	// 创建归档存储过程（如果不存在）
	createArchiveFunction(db)

	// 初始化 Redis (可选)
	var rdb *redis.Client
	if c.Redis.Host != "" {
		var err error
		rdb, err = redisClient.NewRedisClient(c.Redis.Addr(), c.Redis.Password, c.Redis.DB)
		if err != nil {
			// Redis 连接失败只打印警告，不中断启动
			println("Warning: Failed to connect to Redis:", err.Error())
		} else {
			println("Redis connected successfully")
		}
	}

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

	// 初始化归档任务
	archiveTask := tasks.NewArchiveTask(db, c.Archive.RetentionDay, c.Archive.Enabled)
	if c.Archive.Enabled {
		go archiveTask.Start(context.Background())
	}

	// 初始化通知管理器
	var notificationMgr notification.NotificationManager
	if c.Notification.Enabled {
		notificationMgr = initNotificationManager(db, rdb, c)
	}

	return &ServiceContext{
		Config:          c,
		DB:              db,
		Redis:           rdb,
		Ingestor:        ingestor,
		ArchiveTask:     archiveTask,
		NotificationMgr: notificationMgr,
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
			Name:          "home",
			Path:          "/home",
			Component:     "layout.base",
			Order:         0,
			Meta:          `{"title":"home","i18nKey":"route.home","icon":"mdi:home"}`,
			RequiredRoles: []string{"admin", "analyst", "viewer"},
		},
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

// createArchiveFunction 创建归档存储过程
func createArchiveFunction(db *gorm2.DB) {
	sql := `
CREATE OR REPLACE FUNCTION archive_old_logs(retention_days INTEGER DEFAULT 90)
RETURNS INTEGER AS $$
DECLARE
    archived_count INTEGER;
    archive_date TIMESTAMP;
BEGIN
    archive_date := NOW() - (retention_days || ' days')::INTERVAL;

    -- 将旧数据移动到归档表
    WITH moved_rows AS (
        DELETE FROM caddy_logs
        WHERE log_time < archive_date
        RETURNING *
    )
    INSERT INTO caddy_logs_archive (id, created_at, updated_at, log_time, country, province, city, host, method, uri, proto, status, size, user_agent, remote_ip, client_ip, raw_log, extra_data)
    SELECT id, created_at, updated_at, log_time, country, province, city, host, method, uri, proto, status, size, user_agent, remote_ip, client_ip, raw_log, extra_data FROM moved_rows;

    GET DIAGNOSTICS archived_count = ROW_COUNT;

    RETURN archived_count;
END;
$$ LANGUAGE plpgsql;
`
	if err := db.Exec(sql).Error; err != nil {
		println("Warning: Failed to create archive function:", err.Error())
	} else {
		println("Archive function created successfully")
	}
}

// initNotificationManager 初始化通知管理器
func initNotificationManager(db *gorm2.DB, rdb *redis.Client, c config.Config) notification.NotificationManager {
	// 创建通知管理器
	mgr := notification.NewManager(db, rdb)

	// 注册通知提供者
	_ = mgr.RegisterProvider(providers.NewWebhookProvider())
	_ = mgr.RegisterProvider(providers.NewEmailProvider())

	// 从配置文件同步通知渠道到数据库
	if len(c.Notification.Channels) > 0 {
		syncChannelsFromConfig(db, c.Notification.Channels)
	}

	// 从配置文件同步告警规则到数据库
	if len(c.Notification.Rules) > 0 {
		syncRulesFromConfig(db, c.Notification.Rules)
	}

	// 启动通知管理器
	if err := mgr.Start(context.Background()); err != nil {
		println("Warning: Failed to start notification manager:", err.Error())
		return nil
	}

	println("Notification manager started successfully")

	// 发送系统启动通知
	event := notification.NewEvent(
		notification.EventSystemStartup,
		notification.LevelInfo,
		"系统启动",
		"LogFlux 系统已成功启动",
	)
	go mgr.Notify(context.Background(), event)

	return mgr
}

// syncChannelsFromConfig 从配置文件同步通知渠道到数据库
func syncChannelsFromConfig(db *gorm2.DB, channels []config.ChannelConf) {
	for _, ch := range channels {
		var existing model.NotificationChannel
		result := db.Where("name = ?", ch.Name).First(&existing)

		if result.Error == gorm2.ErrRecordNotFound {
			// 创建新渠道
			channel := model.NotificationChannel{
				Name:        ch.Name,
				Type:        ch.Type,
				Enabled:     ch.Enabled,
				Config:      model.JSONMap(ch.Config),
				Events:      ch.Events,
				Description: ch.Description,
			}
			if err := db.Create(&channel).Error; err != nil {
				println("Warning: Failed to create notification channel:", ch.Name, err.Error())
			} else {
				println("Created notification channel:", ch.Name)
			}
		} else {
			// 更新现有渠道
			updates := map[string]interface{}{
				"type":        ch.Type,
				"enabled":     ch.Enabled,
				"config":      model.JSONMap(ch.Config),
				"events":      ch.Events,
				"description": ch.Description,
			}
			if err := db.Model(&existing).Updates(updates).Error; err != nil {
				println("Warning: Failed to update notification channel:", ch.Name, err.Error())
			} else {
				println("Updated notification channel:", ch.Name)
			}
		}
	}
}

// syncRulesFromConfig 从配置文件同步告警规则到数据库
func syncRulesFromConfig(db *gorm2.DB, rules []config.RuleConf) {
	for _, r := range rules {
		var existing model.NotificationRule
		result := db.Where("name = ?", r.Name).First(&existing)

		// 将渠道名称转换为渠道 ID
		var channelIDs []int64
		if len(r.ChannelNames) > 0 {
			var channels []model.NotificationChannel
			db.Where("name IN ?", r.ChannelNames).Find(&channels)
			for _, ch := range channels {
				channelIDs = append(channelIDs, int64(ch.ID))
			}
		}

		if result.Error == gorm2.ErrRecordNotFound {
			// 创建新规则
			rule := model.NotificationRule{
				Name:            r.Name,
				Enabled:         r.Enabled,
				RuleType:        r.RuleType,
				EventType:       r.EventType,
				Condition:       model.JSONMap(r.Condition),
				ChannelIDs:      channelIDs,
				Template:        r.Template,
				SilenceDuration: r.SilenceDuration,
				Description:     r.Description,
			}
			if err := db.Create(&rule).Error; err != nil {
				println("Warning: Failed to create notification rule:", r.Name, err.Error())
			} else {
				println("Created notification rule:", r.Name)
			}
		} else {
			// 更新现有规则
			updates := map[string]interface{}{
				"enabled":          r.Enabled,
				"rule_type":        r.RuleType,
				"event_type":       r.EventType,
				"condition":        model.JSONMap(r.Condition),
				"channel_ids":      channelIDs,
				"template":         r.Template,
				"silence_duration": r.SilenceDuration,
				"description":      r.Description,
			}
			if err := db.Model(&existing).Updates(updates).Error; err != nil {
				println("Warning: Failed to update notification rule:", r.Name, err.Error())
			} else {
				println("Updated notification rule:", r.Name)
			}
		}
	}
}
