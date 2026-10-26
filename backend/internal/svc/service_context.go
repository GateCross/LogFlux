package svc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	cronutil "logflux/common/cron"
	"logflux/common/gorm"
	ingest "logflux/common/ingest"
	ingestcaddy "logflux/common/ingest/caddy"
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
	notificationmodel "logflux/model/notification"
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
	CertSyncScheduler *tasks.CertSyncScheduler
	IngestMgr         *ingest.IngestManager
	NotificationMgr   notification.NotificationManager
	ActiveUser        rest.Middleware
	RateLimit         rest.Middleware
	IPRegionCheck     rest.Middleware
	IPRegionMgr       *middleware.IPRegionMiddleware
	SystemConfigModel *systemmodel.SystemConfigModel
	UserModel         usermodel.UserModel
	CronTaskModel     cronmodel.CronTaskModel
	CronTaskFileModel cronmodel.CronTaskFileModel
	CaddyLogModel     caddymodel.CaddyLogModel
	CertificateModel  caddymodel.CertificateModel
	SystemLogModel    ingestmodel.SystemLogModel
	WafEventModel     wafmodel.WafEventModel
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
		&caddymodel.CaddyServer{},
		&caddymodel.CaddyConfigHistory{},
		// 证书中心库存
		&caddymodel.Certificate{},
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
		&wafmodel.WafEvent{},
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
	backfillWafPolicyCRSDefaults(db)
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

	initWafDefaultSources(db)
	initWafDefaultPolicies(db)

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

	// 初始化证书每日同步调度器（执行器在 main 中注入，避免 tasks 依赖 logic 层）
	certSyncScheduler := tasks.NewCertSyncScheduler()

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

	// 初始化日志摄取管理器：WAF 审计日志与站点日志文件都从这里 tail 入库。
	// 站点日志由 Caddy 写入 /var/log/caddy/<name>.log，LogFlux 自身不经过 middleware，
	// 因此必须由采集器读取文件，否则站点级访问/错误日志不会进入 caddy_logs。
	ingestMgr := ingest.NewIngestManager(db)
	ingestMgr.SetCaddyGeoResolver(ipRegionMgr.Resolve)

	// 本机节点先补注册站点日志目录源：重启后即使没有配置变更，
	// 站点日志也能继续被采集，不必等下一次 writer.Apply 触发。
	var localServers []caddymodel.CaddyServer
	db.Where("type = ?", "local").Find(&localServers)
	for _, srv := range localServers {
		ingest.EnsureSiteLogSource(db, srv.ID, logx.WithContext(context.Background()))
	}

	var caddySources []ingestmodel.LogSource
	db.Where("enabled = ? AND type = ?", true, "caddy").Find(&caddySources)
	for _, src := range caddySources {
		ingestMgr.StartSource(src)
	}

	// 存量审计访问日志异步回填为归一化 WAF 事件
	backfillWafEvents(db)

	return &ServiceContext{
		Config:            c,
		DB:                db,
		Redis:             rdb,
		ArchiveTask:       archiveTask,
		CronScheduler:     cronScheduler,
		WafScheduler:      wafScheduler,
		CertSyncScheduler: certSyncScheduler,
		IngestMgr:         ingestMgr,
		NotificationMgr:   notificationMgr,
		ActiveUser:        middleware.NewActiveUserMiddleware(db).Handle,
		RateLimit:         middleware.NewRateLimitMiddleware(5, time.Minute, "/api/login", "/api/refreshToken").Handle, // 登录接口 5 次/分钟
		IPRegionCheck:     ipRegionMgr.Handle,
		IPRegionMgr:       ipRegionMgr,
		SystemConfigModel: systemConfigModel,
		UserModel:         usermodel.NewUserModel(db),
		CronTaskModel:     cronmodel.NewCronTaskModel(db),
		CronTaskFileModel: cronmodel.NewCronTaskFileModel(db),
		CaddyLogModel:     caddymodel.NewCaddyLogModel(db),
		CertificateModel:  caddymodel.NewCertificateModel(db),
		SystemLogModel:    ingestmodel.NewSystemLogModel(db),
		WafEventModel:     wafmodel.NewWafEventModel(db),
	}
}

// backfillWafEvents 把存量审计来源访问日志归一化写入 WAF 事件表；
// 幂等可重跑，按 id 分批推进，失败中止留待下次启动重试。
func backfillWafEvents(db *gorm2.DB) {
	safego.New(context.Background(), "WAF 历史事件回填").Go(func() {
		const batchSize = 500
		var (
			lastID uint
			total  int
		)
		eventModel := wafmodel.NewWafEventModel(db)
		for {
			var logs []caddymodel.CaddyLog
			if err := db.Where("id > ? AND extra_data::text LIKE ?", lastID, `%"source":"waf"%`).
				Order("id asc").Limit(batchSize).Find(&logs).Error; err != nil {
				logx.Errorf("回填 WAF 历史事件查询失败: %v", err)
				return
			}
			if len(logs) == 0 {
				break
			}

			events := make([]*wafmodel.WafEvent, 0, len(logs))
			for i := range logs {
				events = append(events, ingestcaddy.WafEventsFromCaddyLog(&logs[i])...)
			}
			if len(events) > 0 {
				if err := eventModel.SaveIgnoreConflict(context.Background(), events); err != nil {
					logx.Errorf("回填 WAF 历史事件写入失败: %v", err)
					return
				}
				total += len(events)
			}
			lastID = logs[len(logs)-1].ID
		}
		if total > 0 {
			logx.Infof("WAF 历史事件回填完成，共 %d 条", total)
		}
	})
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

// backfillWafPolicyCRSDefaults 把历史策略行的 CRS 参数对齐到新语义。
//
// 升级前写入的阈值是 10/8（旧预设），而旧代码的 PL 变量写错、实际从未生效，
// 所以这些行是「PL 无效 + 阈值宽松」的组合。新代码修好 PL 后若不迁移，
// 点一次「应用到 Caddy」就会把线上已经跑着的 5/4 覆盖回 10/8，防护反而变弱。
//
// 只处理还没设置检测级别的行，跑一次即收敛；之后用户改过的值不会被覆盖。
func backfillWafPolicyCRSDefaults(db *gorm2.DB) {
	if db == nil || db.Dialector == nil || db.Dialector.Name() != "postgres" {
		return
	}
	if !db.Migrator().HasTable(&wafmodel.WafPolicy{}) ||
		!db.Migrator().HasColumn(&wafmodel.WafPolicy{}, "crs_detection_paranoia_level") {
		return
	}

	// 检测级别缺省时按阻断级别补齐，阈值统一回官方推荐值。
	updateSQL := `
UPDATE waf_policies
SET crs_detection_paranoia_level = crs_paranoia_level,
    crs_inbound_anomaly_threshold = 5,
    crs_outbound_anomaly_threshold = 4
WHERE COALESCE(crs_detection_paranoia_level, 0) <= 0
`
	result := db.Exec(updateSQL)
	if result.Error != nil {
		logx.Errorf("回填 WAF 策略 CRS 默认值失败: %v", result.Error)
		return
	}
	if result.RowsAffected > 0 {
		logx.Infof("已回填 %d 条 WAF 策略的 CRS 参数", result.RowsAffected)
	}
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
		CrsDetectionParanoiaLevel:   1,
		CrsInboundAnomalyThreshold:  5,
		CrsOutboundAnomalyThreshold: 4,
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
