package svc

import (
	"context"
	"errors"
	"fmt"
	"logflux/common/gorm"
	"logflux/common/logging"
	redisClient "logflux/common/redis"
	"logflux/internal/config"
	"logflux/internal/ingest"
	"logflux/internal/notification"
	"logflux/internal/notification/providers"
	"logflux/internal/notification/template"
	"logflux/internal/tasks"
	"logflux/model"
	"os"
	"path/filepath"
	"strings"

	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"
	gorm2 "gorm.io/gorm"
)

type ServiceContext struct {
	Config          config.Config
	DB              *gorm2.DB
	Redis           *redis.Client
	Ingestor        *ingest.IngestManager
	ArchiveTask     *tasks.ArchiveTask
	CronScheduler   *tasks.CronScheduler
	WafScheduler    *tasks.WafScheduler
	NotificationMgr notification.NotificationManager
}

func NewServiceContext(c config.Config) *ServiceContext {
	db := gorm.InitGorm(c.Database.DSN())

	// Auto Migrate - 包含归档表和通知表
	db.AutoMigrate(
		&model.User{},
		&model.CaddyLog{},
		&model.CaddyLogArchive{}, // 归档表
		&model.SystemLog{},
		&model.LogIngestCursor{},
		&model.LogSource{},
		&model.Role{},
		&model.Menu{},
		&model.CaddyServer{},
		&model.CaddyConfigHistory{},
		// 通知相关表
		&model.NotificationChannel{},
		&model.NotificationRule{},
		&model.NotificationLog{},
		&model.NotificationJob{},
		&model.NotificationTemplate{},
		// 定时任务表
		&model.CronTask{},
		&model.CronTaskLog{},
		// WAF 更新管理
		&model.WafSource{},
		&model.WafRelease{},
		&model.WafUpdateJob{},
		&model.WafPolicy{},
		&model.WafPolicyRevision{},
		&model.WafRuleExclusion{},
		&model.WafPolicyBinding{},
		&model.WafPolicyFalsePositiveFeedback{},
	)

	initWafWorkspace(&c)

	// 创建归档存储过程（如果不存在）
	createArchiveFunction(db)

	// 后端日志直接写入数据库（异步）
	if writer := logging.NewDBWriter(db, "backend"); writer != nil {
		logx.AddWriter(writer)
	}

	// 初始化 Redis (可选)
	var rdb *redis.Client
	if c.Redis.Host != "" {
		var err error
		rdb, err = redisClient.NewRedisClient(c.Redis.Addr(), c.Redis.Password, c.Redis.DB)
		if err != nil {
			// Redis 连接失败只打印警告，不中断启动
			logx.Errorf("Warning: Failed to connect to Redis: %s", err.Error())
		} else {
			logx.Info("Redis connected successfully")
		}
	}

	// 初始化 RBAC 数据
	initRBACData(db)
	initWafDefaultSources(db)
	initWafDefaultPolicies(db)

	// 初始化默认管理员账号（自动生成随机复杂密码并仅在首次初始化时明文输出）
	ensureAdminUser(db)

	// Init Ingestor
	ingestor := ingest.NewIngestManager(db)

	// Load enabled sources from DB
	var sources []model.LogSource
	db.Where("enabled = ?", true).Find(&sources)
	for _, source := range sources {
		ingestor.StartSource(source)
	}

	// Legacy config support (migration)
	if c.CaddyLogPath != "" {
		// Check if exists in DB, if not add it
		var cnt int64
		db.Model(&model.LogSource{}).Where("path = ?", c.CaddyLogPath).Count(&cnt)
		if cnt == 0 {
			db.Create(&model.LogSource{
				Name:         "Default Config",
				Path:         c.CaddyLogPath,
				Type:         "caddy",
				Enabled:      true,
				ScanInterval: ingest.DefaultScanIntervalSec(),
			})
			ingestor.StartWithInterval(c.CaddyLogPath, ingest.DefaultScanIntervalSec(), "caddy")
		}
	}

	caddyRuntimeLogPath := strings.TrimSpace(c.CaddyRuntimeLogPath)
	if caddyRuntimeLogPath != "" {
		var cnt int64
		db.Model(&model.LogSource{}).Where("path = ?", caddyRuntimeLogPath).Count(&cnt)
		if cnt == 0 {
			db.Create(&model.LogSource{
				Name:         "Caddy Runtime",
				Path:         caddyRuntimeLogPath,
				Type:         "caddy_runtime",
				Enabled:      true,
				ScanInterval: ingest.DefaultScanIntervalSec(),
			})
			ingestor.StartWithInterval(caddyRuntimeLogPath, ingest.DefaultScanIntervalSec(), "caddy_runtime")
		}
	}

	// 初始化通知管理器（仅依赖数据库配置）
	notificationMgr := initNotificationManager(db, rdb)

	// 初始化归档任务
	archiveTask := tasks.NewArchiveTask(db, c.Archive.RetentionDay, c.Archive.Enabled, notificationMgr)
	if c.Archive.Enabled {
		go archiveTask.Start(context.Background())
	}

	// 初始化定时任务调度器
	cronScheduler := tasks.NewCronScheduler(db)
	cronScheduler.Start()

	// 初始化 WAF 更新调度器（执行器在 main 中注入）
	wafScheduler := tasks.NewWafScheduler(db)

	return &ServiceContext{
		Config:          c,
		DB:              db,
		Redis:           rdb,
		Ingestor:        ingestor,
		ArchiveTask:     archiveTask,
		CronScheduler:   cronScheduler,
		WafScheduler:    wafScheduler,
		NotificationMgr: notificationMgr,
	}
}

func initWafWorkspace(c *config.Config) {
	if c == nil {
		return
	}

	configuredDir := strings.TrimSpace(c.Waf.WorkDir)
	securityDir := "/config/security"
	if configuredDir != "" && filepath.Clean(configuredDir) != securityDir {
		logx.Infof("WAF 工作目录仅允许使用安全目录: from=%s to=%s", configuredDir, securityDir)
	}

	c.Waf.WorkDir = securityDir
	if err := ensureWafWorkspaceDirs(securityDir); err != nil {
		logx.Errorf("初始化 WAF 工作目录失败: %s, err=%v", securityDir, err)
		logx.Errorf("WAF 工作目录初始化失败，后续涉及文件操作将报错: workDir=%s", c.Waf.WorkDir)
		return
	}

	logx.Infof("WAF 工作目录已初始化: %s", fmt.Sprintf("%s/{packages,releases}", securityDir))
}

func ensureWafWorkspaceDirs(baseDir string) error {
	trimmed := strings.TrimSpace(baseDir)
	if trimmed == "" {
		return fmt.Errorf("workdir is empty")
	}

	subDirs := []string{"", "packages", "releases"}
	for _, subDir := range subDirs {
		target := filepath.Join(trimmed, subDir)
		if err := os.MkdirAll(target, 0o755); err != nil {
			return fmt.Errorf("create dir failed: %s, %w", target, err)
		}
	}

	return nil
}

func initWafDefaultSources(db *gorm2.DB) {
	var total int64
	if err := db.Model(&model.WafSource{}).Count(&total).Error; err != nil {
		logx.Errorf("统计 WAF 源数量失败: %v", err)
		return
	}

	if total > 0 {
		return
	}

	defaultSources := []model.WafSource{
		{
			Name:         "default-crs",
			Kind:         "crs",
			Mode:         "remote",
			URL:          "https://codeload.github.com/coreruleset/coreruleset/tar.gz/refs/heads/main",
			ChecksumURL:  "",
			ProxyURL:     "",
			AuthType:     "none",
			AuthSecret:   "",
			Schedule:     "0 0 */6 * * *",
			Enabled:      true,
			AutoCheck:    true,
			AutoDownload: true,
			AutoActivate: false,
			Meta: model.JSONMap{
				"default":  true,
				"official": true,
				"repo":     "https://github.com/coreruleset/coreruleset",
			},
		},
		{
			Name:         "official-crs",
			Kind:         "crs",
			Mode:         "remote",
			URL:          "https://github.com/coreruleset/coreruleset/archive/refs/heads/main.tar.gz",
			ChecksumURL:  "",
			ProxyURL:     "",
			AuthType:     "none",
			AuthSecret:   "",
			Schedule:     "0 0 */6 * * *",
			Enabled:      true,
			AutoCheck:    true,
			AutoDownload: true,
			AutoActivate: false,
			Meta: model.JSONMap{
				"official": true,
				"repo":     "https://github.com/coreruleset/coreruleset",
			},
		},
	}

	for i := range defaultSources {
		source := defaultSources[i]
		var existing model.WafSource
		err := db.Where("name = ?", source.Name).First(&existing).Error
		if errors.Is(err, gorm2.ErrRecordNotFound) {
			if createErr := db.Create(&source).Error; createErr != nil {
				logx.Errorf("初始化默认 WAF 源失败: name=%s err=%v", source.Name, createErr)
			}
		} else if err != nil {
			logx.Errorf("查询默认 WAF 源失败: name=%s err=%v", source.Name, err)
		}
	}
}

func initWafDefaultPolicies(db *gorm2.DB) {
	if db == nil {
		return
	}

	var total int64
	if err := db.Model(&model.WafPolicy{}).Count(&total).Error; err != nil {
		logx.Errorf("统计 WAF 策略数量失败: %v", err)
		return
	}
	if total > 0 {
		return
	}

	defaultPolicy := model.WafPolicy{
		Name:                        "default-global-policy",
		Description:                 "默认全局策略",
		Enabled:                     true,
		IsDefault:                   true,
		EngineMode:                  "on",
		AuditEngine:                 "relevantonly",
		AuditLogFormat:              "json",
		AuditRelevantStatus:         "^(?:5|4(?!04))",
		RequestBodyAccess:           true,
		RequestBodyLimit:            10 * 1024 * 1024,
		RequestBodyNoFilesLimit:     1024 * 1024,
		CrsTemplate:                 "low_fp",
		CrsParanoiaLevel:            1,
		CrsInboundAnomalyThreshold:  10,
		CrsOutboundAnomalyThreshold: 8,
		Config: model.JSONMap{
			"scope": "global",
		},
	}

	if err := db.Create(&defaultPolicy).Error; err != nil {
		logx.Errorf("初始化默认 WAF 策略失败: %v", err)
		return
	}

	directives := "SecRuleEngine On\nSecAuditEngine RelevantOnly\nSecAuditLogFormat JSON\nSecAuditLogRelevantStatus ^(?:5|4(?!04))\nSecRequestBodyAccess On\nSecRequestBodyLimit 10485760\nSecRequestBodyNoFilesLimit 1048576\nSecAction \"id:900000,phase:1,pass,nolog,t:none,setvar:tx.paranoia_level=1\"\nSecAction \"id:900110,phase:1,pass,nolog,t:none,setvar:tx.inbound_anomaly_score_threshold=10\"\nSecAction \"id:900100,phase:1,pass,nolog,t:none,setvar:tx.outbound_anomaly_score_threshold=8\""
	revision := model.WafPolicyRevision{
		PolicyID:           defaultPolicy.ID,
		Version:            1,
		Status:             "published",
		ConfigSnapshot:     defaultPolicy.Config,
		DirectivesSnapshot: directives,
		Operator:           "system",
		Message:            "init default policy",
	}
	if err := db.Create(&revision).Error; err != nil {
		logx.Errorf("初始化默认 WAF 策略版本失败: %v", err)
	}

	defaultBinding := model.WafPolicyBinding{
		PolicyID:  defaultPolicy.ID,
		Name:      "default-global-binding",
		Enabled:   true,
		ScopeType: "global",
		Priority:  100,
	}
	if err := db.Where("policy_id = ? AND scope_type = ? AND priority = ?", defaultPolicy.ID, "global", 100).
		FirstOrCreate(&defaultBinding).Error; err != nil {
		logx.Errorf("初始化默认 WAF 策略绑定失败: %v", err)
	}
}

func (svc *ServiceContext) EnsureWafDefaultSources() {
	if svc == nil || svc.DB == nil {
		return
	}
	initWafDefaultSources(svc.DB)
}

func (svc *ServiceContext) EnsureWafEngineDefaultSource() {
	return
}

// initRBACData 初始化 RBAC 角色和菜单数据
func initRBACData(db *gorm2.DB) {
	// 初始化默认角色
	roles := []model.Role{
		{
			Name:        "admin",
			DisplayName: "管理员",
			Description: "系统管理员，拥有所有权限",
			Permissions: []string{
				"dashboard", "manage", "manage_user", "manage_role", "manage_menu",
				"logs", "logs_caddy", "security",
				"notification", "notification_channel", "notification_rule", "notification_template", "notification_log",
				"user_center",
			},
		},
		{
			Name:        "analyst",
			DisplayName: "分析师",
			Description: "数据分析师，可访问系统日志（logs）与 Caddy 访问日志（logs_caddy）",
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
		var existingRole model.Role
		if db.Where("name = ?", role.Name).First(&existingRole).Error == gorm2.ErrRecordNotFound {
			db.Create(&role)
		} else {
			db.Model(&existingRole).Select("DisplayName", "Description", "Permissions").Updates(role)
		}
	}

	// 初始化菜单数据
	menus := []model.Menu{
		{
			Name:          "dashboard",
			Path:          "/dashboard",
			Component:     "layout.base$view.dashboard",
			Order:         1,
			Meta:          `{"title":"dashboard","i18nKey":"route.dashboard","icon":"mdi:monitor-dashboard","order":1}`,
			RequiredRoles: []string{}, // Public
		},
		{
			Name:          "caddy",
			Path:          "/caddy",
			Component:     "layout.base",
			Order:         2,
			Meta:          `{"title":"caddy","i18nKey":"route.caddy","icon":"carbon:cloud-monitoring","order":2}`,
			RequiredRoles: []string{"admin", "analyst"},
		},
		{
			Name:          "security",
			Path:          "/security",
			Component:     "layout.base$view.security",
			Order:         3,
			Meta:          `{"title":"security","i18nKey":"route.security","icon":"carbon:security","order":3}`,
			RequiredRoles: []string{"admin"},
		},
		{
			Name:          "manage",
			Path:          "/manage",
			Component:     "layout.base",
			Order:         9,
			Meta:          `{"title":"manage","i18nKey":"route.manage","icon":"carbon:cloud-service-management","order":9,"roles":["admin"]}`,
			RequiredRoles: []string{"admin"},
		},
		{
			Name:          "notification",
			Path:          "/notification",
			Component:     "layout.base",
			Order:         10,
			Meta:          `{"title":"notification","i18nKey":"route.notification","icon":"carbon:notification","order":10,"roles":["admin"]}`,
			RequiredRoles: []string{"admin"},
		},
		// --- 子菜单 ---
		{
			Name:          "caddy_config",
			Path:          "/caddy/config",
			Component:     "view.caddy_config",
			Meta:          `{"title":"caddy_config","i18nKey":"route.caddy_config","icon":"carbon:settings"}`,
			RequiredRoles: []string{"admin", "analyst"},
		},
		{
			Name:          "caddy_log",
			Path:          "/caddy/log",
			Component:     "view.caddy_log",
			Meta:          `{"title":"caddy_log","i18nKey":"route.caddy_log","icon":"carbon:catalog"}`,
			RequiredRoles: []string{"admin", "analyst"},
		},
		{
			Name:          "caddy_system_log",
			Path:          "/caddy/system-log",
			Component:     "view.caddy_system-log",
			Meta:          `{"title":"caddy_system-log","i18nKey":"route.caddy_system-log","icon":"carbon:terminal"}`,
			RequiredRoles: []string{"admin", "analyst"},
		},
		{
			Name:          "caddy_source",
			Path:          "/caddy/source",
			Component:     "view.caddy_source",
			Meta:          `{"title":"caddy_source","i18nKey":"route.caddy_source","icon":"carbon:data-base"}`,
			RequiredRoles: []string{"admin", "analyst"},
		},
		{
			Name:          "manage_user",
			Path:          "/manage/user",
			Component:     "view.manage_user",
			Meta:          `{"title":"manage_user","i18nKey":"route.manage_user","icon":"ic:round-manage-accounts","roles":["admin"]}`,
			RequiredRoles: []string{"admin"},
		},
		{
			Name:          "manage_role",
			Path:          "/manage/role",
			Component:     "view.manage_role",
			Meta:          `{"title":"manage_role","i18nKey":"route.manage_role","icon":"carbon:user-role","roles":["admin"]}`,
			RequiredRoles: []string{"admin"},
		},
		{
			Name:          "manage_menu",
			Path:          "/manage/menu",
			Component:     "view.manage_menu",
			Meta:          `{"title":"manage_menu","i18nKey":"route.manage_menu","icon":"material-symbols:menu-book","roles":["admin"]}`,
			RequiredRoles: []string{"admin"},
		},
		{
			Name:          "notification_channel",
			Path:          "/notification/channel",
			Component:     "view.notification_channel",
			Meta:          `{"title":"notification_channel","i18nKey":"route.notification_channel","icon":"mdi:broadcast","roles":["admin"]}`,
			RequiredRoles: []string{"admin"},
		},
		{
			Name:          "notification_rule",
			Path:          "/notification/rule",
			Component:     "view.notification_rule",
			Meta:          `{"title":"notification_rule","i18nKey":"route.notification_rule","icon":"carbon:rule","roles":["admin"]}`,
			RequiredRoles: []string{"admin"},
		},
		{
			Name:          "notification_template",
			Path:          "/notification/template",
			Component:     "view.notification_template",
			Meta:          `{"title":"notification_template","i18nKey":"route.notification_template","icon":"carbon:template","roles":["admin"]}`,
			RequiredRoles: []string{"admin"},
		},
		{
			Name:          "notification_log",
			Path:          "/notification/log",
			Component:     "view.notification_log",
			Meta:          `{"title":"notification_log","i18nKey":"route.notification_log","icon":"carbon:script","roles":["admin"]}`,
			RequiredRoles: []string{"admin"},
		},
		{
			Name:          "cron",
			Path:          "/cron",
			Component:     "layout.base$view.cron",
			Order:         5,
			Meta:          `{"title":"cron","i18nKey":"route.cron","icon":"mdi:clock-time-four-outline","order":5,"roles":["admin"]}`,
			RequiredRoles: []string{"admin"},
		},
		// --- 个人中心 ---
		{
			Name:          "user",
			Path:          "/user",
			Component:     "layout.base",
			Order:         11,
			Meta:          `{"title":"user","i18nKey":"route.user","icon":"carbon:user-avatar","order":11}`,
			RequiredRoles: []string{}, // Public
		},
		{
			Name:          "user_center",
			Path:          "/user/center",
			Component:     "view.user_center",
			Meta:          `{"title":"user_center","i18nKey":"route.user_center","icon":"carbon:user-profile"}`,
			RequiredRoles: []string{}, // Public
		},
	}

	// 第一步：确保所有菜单存在
	createdMenus := make(map[string]bool)
	for i := range menus {
		menu := menus[i]
		var existingMenu model.Menu
		if db.Where("name = ?", menu.Name).First(&existingMenu).Error == gorm2.ErrRecordNotFound {
			db.Create(&menu)
			createdMenus[menu.Name] = true
		} else {
			// 只更新代码路径相关的技术字段，保留用户的配置（排序、图标、权限等）
			db.Model(&existingMenu).Select("Path", "Component").Updates(menu)
		}
	}

	// 兼容历史菜单数据：统一系统日志菜单的组件与 i18nKey
	db.Model(&model.Menu{}).Where("name = ?", "caddy_system_log").Updates(map[string]interface{}{
		"component": "view.caddy_system-log",
		"meta":      `{"title":"caddy_system-log","i18nKey":"route.caddy_system-log","icon":"carbon:terminal"}`,
	})

	// 第二步：建立父子关系
	setParent := func(childName, parentName string) {
		// 仅对新创建的菜单设置默认父级，避免覆盖用户的层级调整
		if !createdMenus[childName] {
			return
		}

		var child, parent model.Menu
		if db.Where("name = ?", childName).First(&child).Error == nil &&
			db.Where("name = ?", parentName).First(&parent).Error == nil {
			db.Model(&child).Update("parent_id", parent.ID)
		}
	}

	setParentForce := func(childName, parentName string) {
		var child, parent model.Menu
		if db.Where("name = ?", childName).First(&child).Error == nil &&
			db.Where("name = ?", parentName).First(&parent).Error == nil {
			db.Model(&child).Update("parent_id", parent.ID)
		}
	}

	setParent("caddy_config", "caddy")
	setParent("caddy_log", "caddy")
	setParent("caddy_source", "caddy")
	setParent("manage_user", "manage")
	setParent("manage_role", "manage")
	setParent("manage_menu", "manage")
	setParent("notification_channel", "notification")
	setParent("notification_rule", "notification")
	setParent("notification_template", "notification")
	setParent("notification_template", "notification")
	setParent("notification_log", "notification")
	setParent("notification_log", "notification")
	setParent("user_center", "user")
	setParentForce("caddy_system_log", "manage")
	// setParent("cron", "manage") // moved to top level

	// 清理遗留数据
	db.Where("name = ?", "home").Delete(&model.Menu{})
	db.Where("path = ?", "/home").Delete(&model.Menu{})
	db.Where("component = ?", "home").Delete(&model.Menu{})
	db.Where("name = ?", "caddy_waf").Delete(&model.Menu{})
	db.Where("name in ?", []string{"waf", "crs"}).Delete(&model.Menu{})
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
		logx.Errorf("Warning: Failed to create archive function: %s", err.Error())
	} else {
		logx.Info("Archive function created successfully")
	}
}

// initNotificationManager 初始化通知管理器（仅从数据库加载配置）
func initNotificationManager(db *gorm2.DB, rdb *redis.Client) notification.NotificationManager {
	// 初始化模板管理器
	templateMgr := template.NewTemplateManager(db)
	// 加载模板 (忽略错误，因为初始可能为空)
	_ = templateMgr.LoadTemplates()

	// 创建通知管理器
	mgr := notification.NewManager(db, rdb, templateMgr)

	// 注册通知提供者
	_ = mgr.RegisterProvider(providers.NewWebhookProvider())
	_ = mgr.RegisterProvider(providers.NewEmailProvider())
	_ = mgr.RegisterProvider(providers.NewTelegramProvider())
	_ = mgr.RegisterProvider(providers.NewSlackProvider())
	_ = mgr.RegisterProvider(providers.NewWeComProvider())
	_ = mgr.RegisterProvider(providers.NewWeChatMPProvider())
	_ = mgr.RegisterProvider(providers.NewDiscordProvider())
	_ = mgr.RegisterProvider(providers.NewInAppProvider())

	// 启动通知管理器（渠道/规则均从数据库加载）
	if err := mgr.Start(context.Background()); err != nil {
		logx.Errorf("Warning: Failed to start notification manager: %s", err.Error())
		return nil
	}

	logx.Info("Notification manager started successfully")

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
