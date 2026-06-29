package svc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	cronutil "logflux/common/cron"
	"logflux/common/gorm"
	ingest "logflux/common/ingest"
	"logflux/common/logging"
	redisClient "logflux/common/redis"
	"logflux/internal/config"
	"logflux/internal/middleware"
	"logflux/internal/notification"
	"logflux/internal/notification/providers"
	"logflux/internal/notification/template"
	"logflux/internal/tasks"
	"logflux/internal/utils/safego"
	caddymodel "logflux/model/caddy"
	commonmodel "logflux/model/common"
	cronmodel "logflux/model/cron"
	ingestmodel "logflux/model/ingest"
	menumodel "logflux/model/menu"
	notificationmodel "logflux/model/notification"
	rolemodel "logflux/model/role"
	systemmodel "logflux/model/system"
	usermodel "logflux/model/user"
	wafmodel "logflux/model/waf"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest"
	gorm2 "gorm.io/gorm"
)

type ServiceContext struct {
	Config            config.Config
	DB                *gorm2.DB
	Redis             *redis.Client
	ArchiveTask       *tasks.ArchiveTask
	CronScheduler     *tasks.CronScheduler
	WafScheduler      *tasks.WafScheduler
	IngestMgr         *ingest.IngestManager
	NotificationMgr   notification.NotificationManager
	Permission        rest.Middleware
	RateLimit         rest.Middleware
	IPRegionCheck     rest.Middleware
	IPRegionMgr       *middleware.IPRegionMiddleware
	SystemConfigModel *systemmodel.SystemConfigModel
	UserModel         usermodel.UserModel
	RoleModel         rolemodel.RoleModel
	MenuModel         menumodel.MenuModel
	CronTaskModel     cronmodel.CronTaskModel
	CronTaskFileModel cronmodel.CronTaskFileModel
	CaddyLogModel     caddymodel.CaddyLogModel
	SystemLogModel    ingestmodel.SystemLogModel
}

func NewServiceContext(c config.Config) *ServiceContext {
	db := gorm.InitGorm(c.Database.DSN())

	// Auto Migrate - 包含归档表和通知表
	db.AutoMigrate(
		&usermodel.User{},
		&caddymodel.CaddyLog{},
		&caddymodel.CaddyLogArchive{}, // 归档表
		&ingestmodel.SystemLog{},
		&ingestmodel.LogIngestCursor{},
		&ingestmodel.LogSource{},
		&rolemodel.Role{},
		&menumodel.Menu{},
		&caddymodel.CaddyServer{},
		&caddymodel.CaddyConfigHistory{},
		// 通知相关表
		&notificationmodel.NotificationChannel{},
		&notificationmodel.NotificationRule{},
		&notificationmodel.NotificationLog{},
		&notificationmodel.NotificationJob{},
		&notificationmodel.NotificationTemplate{},
		// 定时任务表
		&cronmodel.CronTask{},
		&cronmodel.CronTaskLog{},
		&cronmodel.CronTaskFile{},
		// WAF 更新管理
		&wafmodel.WafSource{},
		&wafmodel.WafRelease{},
		&wafmodel.WafUpdateJob{},
		&wafmodel.WafPolicy{},
		// 系统配置
		&systemmodel.SystemConfig{},
	)

	// 处理破坏性变更: 移除 notification_rules 表被废弃的 event_type 列
	if db.Migrator().HasColumn(&notificationmodel.NotificationRule{}, "event_type") {
		err := db.Migrator().DropColumn(&notificationmodel.NotificationRule{}, "event_type")
		if err != nil {
			logx.Errorf("移除废弃列 event_type 失败: %v", err)
		} else {
			logx.Info("已成功移除 notification_rules 表的 event_type 列")
		}
	}

	ensureCronTaskLogRetentionConstraints(db)
	backfillCronTaskLogTaskNames(db)
	initWafWorkspace(&c)
	initCronWorkspace(&c)

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
			logx.Errorf("警告: 连接 Redis 失败: %s", err.Error())
		} else {
			logx.Info("Redis 连接成功")
		}
	}

	// 初始化 RBAC 数据
	initRBACData(db)
	initWafDefaultSources(db)
	initWafDefaultPolicies(db)

	// 初始化默认管理员账号（自动生成随机复杂密码并仅在首次初始化时明文输出）
	ensureAdminUser(db)

	// 初始化通知管理器（仅依赖数据库配置）
	notificationMgr := initNotificationManager(db, rdb)

	// 初始化归档任务
	archiveTask := tasks.NewArchiveTask(db, c.Archive.RetentionDay, c.Archive.Enabled, notificationMgr)
	if c.Archive.Enabled {
		safego.New(context.Background(), "日志归档任务").Go(func() {
			archiveTask.Start(context.Background())
		})
	}

	// 初始化定时任务调度器
	cronScheduler := tasks.NewCronScheduler(db)
	cronScheduler.SetExecutor(tasks.NewCronTaskExecutor(db, c.CronFilesDir))
	cronScheduler.Start()

	// 初始化 WAF 更新调度器（执行器在 main 中注入）
	wafScheduler := tasks.NewWafScheduler(db)

	// 初始化系统配置模型
	systemConfigModel := systemmodel.NewSystemConfigModel(db)

	// 初始化 IP 区域中间件（支持从 DB 加载配置热重载 + 访问日志写入）
	ipRegionMgr := middleware.NewIPRegionMiddleware(c.IPRegion.Enabled, c.IPRegion.AllowList, db)
	if cfg, err := systemConfigModel.GetByKey("ip_region"); err == nil {
		var ipCfg struct {
			Enabled   bool     `json:"enabled"`
			AllowList []string `json:"allowList"`
		}
		if json.Unmarshal([]byte(cfg.Value), &ipCfg) == nil {
			ipRegionMgr.Reload(ipCfg.Enabled, ipCfg.AllowList)
		}
	}

	// 初始化日志摄取管理器（tail WAF 审计日志并入库）
	// 注意：access.log 由 IPRegionMiddleware 直接写入 DB，无需 tail；仅 WAF 审计日志需要 tail
	ingestMgr := ingest.NewIngestManager(db)
	ingestMgr.SetCaddyGeoResolver(ipRegionMgr.Resolve)
	var wafSources []ingestmodel.LogSource
	db.Where("enabled = ? AND name = ?", true, "WAF 审计日志").Find(&wafSources)
	for _, src := range wafSources {
		ingestMgr.StartSource(src)
	}

	return &ServiceContext{
		Config:            c,
		DB:                db,
		Redis:             rdb,
		ArchiveTask:       archiveTask,
		CronScheduler:     cronScheduler,
		WafScheduler:      wafScheduler,
		IngestMgr:         ingestMgr,
		NotificationMgr:   notificationMgr,
		Permission:        middleware.NewPermissionMiddleware(db).Handle,
		RateLimit:         middleware.NewRateLimitMiddleware(5, time.Minute, "/api/login", "/api/refreshToken").Handle, // 登录接口 5 次/分钟
		IPRegionCheck:     ipRegionMgr.Handle,
		IPRegionMgr:       ipRegionMgr,
		SystemConfigModel: systemConfigModel,
		UserModel:         usermodel.NewUserModel(db),
		RoleModel:         rolemodel.NewRoleModel(db),
		MenuModel:         menumodel.NewMenuModel(db),
		CronTaskModel:     cronmodel.NewCronTaskModel(db),
		CronTaskFileModel: cronmodel.NewCronTaskFileModel(db),
		CaddyLogModel:     caddymodel.NewCaddyLogModel(db),
		SystemLogModel:    ingestmodel.NewSystemLogModel(db),
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

func initCronWorkspace(c *config.Config) {
	if c == nil {
		return
	}

	configuredDir := strings.TrimSpace(c.CronFilesDir)
	if configuredDir == "" {
		configuredDir = cronutil.DefaultFilesDir
	}

	c.CronFilesDir = configuredDir
	if err := cronutil.EnsureWorkspace(configuredDir); err != nil {
		logx.Errorf("初始化定时任务脚本目录失败: %s, err=%v", configuredDir, err)
		return
	}

	logx.Infof("定时任务脚本目录已初始化: %s", fmt.Sprintf("%s/{tasks,staging}", configuredDir))
}

func ensureCronTaskLogRetentionConstraints(db *gorm2.DB) {
	if db == nil || db.Dialector == nil || db.Dialector.Name() != "postgres" {
		return
	}

	type fkConstraint struct {
		Name string `gorm:"column:constraint_name"`
	}

	var constraints []fkConstraint
	query := `
SELECT con.conname AS constraint_name
FROM pg_constraint con
JOIN pg_class rel ON rel.oid = con.conrelid
JOIN pg_class ref ON ref.oid = con.confrelid
WHERE con.contype = 'f'
  AND rel.relname = 'cron_task_logs'
  AND ref.relname = 'cron_tasks'
`
	if err := db.Raw(query).Scan(&constraints).Error; err != nil {
		logx.Errorf("查询 Cron 日志外键约束失败: %v", err)
		return
	}

	for _, constraint := range constraints {
		name := strings.TrimSpace(constraint.Name)
		if name == "" {
			continue
		}
		dropSQL := fmt.Sprintf(`ALTER TABLE "cron_task_logs" DROP CONSTRAINT IF EXISTS %s`, quoteIdentifier(name))
		if err := db.Exec(dropSQL).Error; err != nil {
			logx.Errorf("删除 Cron 日志外键约束失败: %s err=%v", name, err)
		} else {
			logx.Infof("已移除 Cron 日志外键约束: %s", name)
		}
	}
}

func backfillCronTaskLogTaskNames(db *gorm2.DB) {
	if db == nil || db.Dialector == nil || db.Dialector.Name() != "postgres" {
		return
	}
	if !db.Migrator().HasTable(&cronmodel.CronTaskLog{}) || !db.Migrator().HasColumn(&cronmodel.CronTaskLog{}, "task_name") {
		return
	}

	updateSQL := `
UPDATE cron_task_logs AS log
SET task_name = task.name
FROM cron_tasks AS task
WHERE log.task_id = task.id
  AND COALESCE(log.task_name, '') = ''
`
	if err := db.Exec(updateSQL).Error; err != nil {
		logx.Errorf("回填 Cron 日志任务名称失败: %v", err)
	}
}

func quoteIdentifier(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

func ensureWafWorkspaceDirs(baseDir string) error {
	trimmed := strings.TrimSpace(baseDir)
	if trimmed == "" {
		return fmt.Errorf("工作目录为空")
	}

	subDirs := []string{"", "packages", "releases"}
	for _, subDir := range subDirs {
		target := filepath.Join(trimmed, subDir)
		if err := os.MkdirAll(target, 0o755); err != nil {
			return fmt.Errorf("创建目录失败: %s, %w", target, err)
		}
	}

	return nil
}

func initWafDefaultSources(db *gorm2.DB) {
	var total int64
	if err := db.Model(&wafmodel.WafSource{}).Count(&total).Error; err != nil {
		logx.Errorf("统计 WAF 源数量失败: %v", err)
		return
	}

	if total > 0 {
		return
	}

	defaultSources := []wafmodel.WafSource{
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
			Meta: commonmodel.JSONMap{
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
			Meta: commonmodel.JSONMap{
				"official": true,
				"repo":     "https://github.com/coreruleset/coreruleset",
			},
		},
	}

	for i := range defaultSources {
		source := defaultSources[i]
		var existing wafmodel.WafSource
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
	if err := db.Model(&wafmodel.WafPolicy{}).Count(&total).Error; err != nil {
		logx.Errorf("统计 WAF 策略数量失败: %v", err)
		return
	}
	if total > 0 {
		return
	}

	defaultPolicy := wafmodel.WafPolicy{
		Name:                        "default-global-policy",
		Description:                 "默认全局策略",
		Enabled:                     true,
		IsDefault:                   true,
		EngineMode:                  "detectiononly",
		AuditEngine:                 "relevantonly",
		AuditLogFormat:              "json",
		AuditRelevantStatus:         wafmodel.DefaultAuditRelevantStatus,
		RequestBodyAccess:           true,
		RequestBodyLimit:            10 * 1024 * 1024,
		RequestBodyNoFilesLimit:     1024 * 1024,
		CrsTemplate:                 "low_fp",
		CrsParanoiaLevel:            1,
		CrsInboundAnomalyThreshold:  10,
		CrsOutboundAnomalyThreshold: 8,
		Config: commonmodel.JSONMap{
			"scope": "global",
		},
	}

	if err := db.Create(&defaultPolicy).Error; err != nil {
		logx.Errorf("初始化默认 WAF 策略失败: %v", err)
		return
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
	roles := []rolemodel.Role{
		{
			Name:        "admin",
			DisplayName: "管理员",
			Description: "系统管理员，拥有所有权限",
			Permissions: []string{
				"dashboard",
				"caddy", "caddy_config", "caddy_access", "logs", "logs_caddy",
				"cron",
				"manage", "manage_user", "manage_role", "manage_menu",
				"notification", "notification_channel", "notification_rule", "notification_template", "notification_log",
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
		var existingRole rolemodel.Role
		if db.Where("name = ?", role.Name).First(&existingRole).Error == gorm2.ErrRecordNotFound {
			db.Create(&role)
		} else {
			db.Model(&existingRole).Select("DisplayName", "Description", "Permissions").Updates(role)
		}
	}

	// 初始化菜单数据
	menus := []menumodel.Menu{
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
			Name:          "caddy_access",
			Path:          "/caddy/access",
			Component:     "view.caddy_access",
			Meta:          `{"title":"caddy_access","i18nKey":"route.caddy_access","icon":"carbon:network-4"}`,
			RequiredRoles: []string{"admin"},
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
			Path:          "/user/center",
			Component:     "layout.base$view.user_center",
			Order:         11,
			Meta:          `{"title":"user","i18nKey":"route.user","icon":"carbon:user-avatar","order":11}`,
			RequiredRoles: []string{}, // Public
		},
	}

	// 第一步：确保所有菜单存在
	createdMenus := make(map[string]bool)
	for i := range menus {
		menu := menus[i]
		var existingMenu menumodel.Menu
		if db.Where("name = ?", menu.Name).First(&existingMenu).Error == gorm2.ErrRecordNotFound {
			db.Create(&menu)
			createdMenus[menu.Name] = true
		} else {
			// 只更新代码路径相关的技术字段，保留用户的配置（排序、图标、权限等）
			db.Model(&existingMenu).Select("Path", "Component").Updates(menu)
		}
	}

	// 兼容历史菜单数据：统一系统日志菜单的组件与 i18nKey
	db.Model(&menumodel.Menu{}).Where("name = ?", "caddy_system_log").Updates(map[string]interface{}{
		"component": "view.caddy_system-log",
		"meta":      `{"title":"caddy_system-log","i18nKey":"route.caddy_system-log","icon":"carbon:terminal"}`,
	})

	// 第二步：建立父子关系
	setParent := func(childName, parentName string) {
		// 仅对新创建的菜单设置默认父级，避免覆盖用户的层级调整
		if !createdMenus[childName] {
			return
		}

		var child, parent menumodel.Menu
		if db.Where("name = ?", childName).First(&child).Error == nil &&
			db.Where("name = ?", parentName).First(&parent).Error == nil {
			db.Model(&child).Update("parent_id", parent.ID)
		}
	}

	setParentForce := func(childName, parentName string) {
		var child, parent menumodel.Menu
		if db.Where("name = ?", childName).First(&child).Error == nil &&
			db.Where("name = ?", parentName).First(&parent).Error == nil {
			db.Model(&child).Update("parent_id", parent.ID)
		}
	}

	setParent("caddy_config", "caddy")
	setParent("caddy_log", "caddy")
	setParent("manage_user", "manage")
	setParent("manage_role", "manage")
	setParent("manage_menu", "manage")
	setParent("notification_channel", "notification")
	setParent("notification_rule", "notification")
	setParent("notification_template", "notification")
	setParent("notification_log", "notification")
	setParentForce("caddy_system_log", "manage")
	setParentForce("caddy_access", "caddy")
	// setParent("cron", "manage") // moved to top level

	// 清理遗留数据
	db.Where("name = ?", "home").Delete(&menumodel.Menu{})
	db.Where("path = ?", "/home").Delete(&menumodel.Menu{})
	db.Where("component = ?", "home").Delete(&menumodel.Menu{})
	db.Where("name = ?", "caddy_source").Delete(&menumodel.Menu{})
	db.Where("name = ?", "caddy_waf").Delete(&menumodel.Menu{})
	db.Where("name in ?", []string{"security", "waf", "waf_security", "crs", "security_access"}).Delete(&menumodel.Menu{})
	db.Where("path = ? OR path LIKE ?", "/security", "/security/%").Delete(&menumodel.Menu{})
	// 修复被误改 name 的 caddy_access 菜单（name 必须是路由 key，不是中文标题）
	db.Model(&menumodel.Menu{}).Where("path = ? AND name != ?", "/caddy/access", "caddy_access").Delete(&menumodel.Menu{})
	// 个人中心现在是一级菜单，清理旧的二级重复菜单并确保父级为空。
	db.Where("name = ?", "user_center").Delete(&menumodel.Menu{})
	db.Model(&menumodel.Menu{}).Where("name = ?", "user").Update("parent_id", nil)
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
		logx.Errorf("警告: 创建归档函数失败: %s", err.Error())
	} else {
		logx.Info("归档函数创建成功")
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
		logx.Errorf("警告: 启动通知管理器失败: %s", err.Error())
		return nil
	}

	logx.Info("通知管理器启动成功")

	// 发送系统启动通知
	event := notification.NewEvent(
		notification.EventSystemStartup,
		notification.LevelInfo,
		"系统启动",
		"LogFlux 系统已成功启动",
	)
	safego.New(context.Background(), "系统启动通知").Go(func() {
		if err := mgr.Notify(context.Background(), event); err != nil {
			logx.Errorf("发送系统启动通知失败: %v", err)
		}
	})

	return mgr
}
