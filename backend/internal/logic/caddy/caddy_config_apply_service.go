package caddy

import (
	"context"
	"fmt"
	caddymodel "logflux/model/caddy"
	"strings"
	"sync"

	ingest "logflux/common/ingest"
	"logflux/internal/svc"
	"logflux/internal/utils/safego"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

// caddyConfigWriteMu 串行化所有 Caddy 配置写路径。
//
// adapt → load → persist 必须作为整体执行：若无互斥，两个并发写请求交错
// （A.load → B.load → B.persist → A.persist）会出现运行态与库内配置分叉，
// 且双方都返回成功；配置写频率极低，全局单锁即可。
var caddyConfigWriteMu sync.Mutex

type caddyConfigApplyService struct {
	svcCtx *svc.ServiceContext
	logger logx.Logger
}

func newCaddyConfigApplyService(svcCtx *svc.ServiceContext, logger logx.Logger) *caddyConfigApplyService {
	return &caddyConfigApplyService{svcCtx: svcCtx, logger: logger}
}

// loadCurrent 读取当前生效配置（存于 system_configs，单实例下只有一份）。
func (s *caddyConfigApplyService) loadCurrent() (string, string, error) {
	return loadCurrentCaddyConfig(s.svcCtx.DB)
}

// checkBaseHash 做乐观并发检查。
//
// baseHash 为空表示旧版客户端，保持向后兼容直接放行；非空时与库内当前哈希比对，
// 不一致说明期间有人改过配置，继续写会静默覆盖对方的改动。
func checkBaseHash(db *gorm.DB, baseHash string) error {
	baseHash = strings.TrimSpace(baseHash)
	if baseHash == "" || db == nil {
		return nil
	}
	current, _ := caddymodel.LoadConfig(db)
	if hashConfig(current) != baseHash {
		return fmt.Errorf("配置已被他人修改，请刷新后重试")
	}
	return nil
}

// configWriteOptions 表达一次配置写入的全部差异，供三条写路径共用同一套语义。
type configWriteOptions struct {
	// history 表 action 名，必填。
	HistoryAction string
	// 额外写一条快照历史，记录写入前的旧配置（简单 WAF 的 last_good 依赖）。
	SnapshotBefore bool
	// 非空时做乐观并发校验，与库内哈希不一致则拒绝写入。
	BaseHash string
	// 与配置同事务执行的落库动作（如简单 WAF 的策略保存），任一步失败整体回滚。
	InTx func(tx *gorm.DB) error
}

// Apply 按「校验 → 加载 → 落库」固定顺序应用配置，是所有 Caddy 配置写路径的唯一入口，
// 全程持有 caddyConfigWriteMu 串行执行。
//
// 顺序统一的原因：/load 本身失败会自动回滚运行态，落库前出错等于什么都没改；
// 只有落库失败才需要把运行态改回去，而那次恢复依赖上一份配置，所以先加载再落库
// 是唯一能把"哪一步失败、要回滚什么"说清楚的顺序。
func (s *caddyConfigApplyService) Apply(server *caddymodel.CaddyServer, config, modules string, opts configWriteOptions) error {
	if server == nil {
		return fmt.Errorf("Caddy 服务器不存在")
	}
	if strings.TrimSpace(opts.HistoryAction) == "" {
		return fmt.Errorf("配置写入缺少 history 动作名")
	}

	caddyConfigWriteMu.Lock()
	defer caddyConfigWriteMu.Unlock()

	// 锁内重读库内最新配置再校验 baseHash：请求开始时的内存快照不可信，
	// 从加载到落库之间隔着 adapt/load 两次带重试的 HTTP 调用（窗口可达数十秒），
	// 期间他人落库的修改若不在此重新比对，会被携带"当时合法"哈希的后写者覆盖。
	if err := checkBaseHash(s.svcCtx.DB, opts.BaseHash); err != nil {
		return err
	}
	previousConfig, previousModules := caddymodel.LoadConfig(s.svcCtx.DB)

	normalizedModules := normalizeCaddyModulesJSON(modules)

	// adapt 前合并全局选项与 :80 入口块模板：库里单独保存的设置在每次应用时
	// 统一收口到最终配置，后续 adapt/load/persist 都以合并结果为准。
	merged, err := s.mergeManagedBlocks(config)
	if err != nil {
		return err
	}
	config = merged

	if err := adaptCaddyfile(server, config); err != nil {
		if s != nil && s.logger != nil {
			s.logger.Errorf("Caddy 配置适配失败: %v", err)
		}
		return err
	}

	if err := loadCaddyfile(server, config); err != nil {
		if s != nil && s.logger != nil {
			s.logger.Errorf("Caddy 配置加载失败: %v", err)
		}
		return err
	}

	if _, err := s.persist(config, normalizedModules, opts, previousConfig, previousModules); err != nil {
		// 未接管过的服务器没有可回退的配置，运行态保持新配置，等下次应用覆盖。
		if strings.TrimSpace(previousConfig) == "" {
			return fmt.Errorf("保存 Caddy 配置到数据库失败: %w", err)
		}
		if rollbackErr := rollbackPolicyConfigToLastGood(server, previousConfig); rollbackErr != nil {
			return fmt.Errorf("保存 Caddy 配置到数据库失败: %v，回滚 Caddy 到旧配置失败: %v", err, rollbackErr)
		}
		return fmt.Errorf("保存 Caddy 配置到数据库失败，已回滚 Caddy 到旧配置: %w", err)
	}

	s.scheduleLogSourceSync(server)
	return nil
}

func (s *caddyConfigApplyService) persist(config, modules string, opts configWriteOptions, previousConfig, previousModules string) (uint, error) {
	var historyID uint
	if err := s.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		if opts.InTx != nil {
			if err := opts.InTx(tx); err != nil {
				return err
			}
		}
		if opts.SnapshotBefore {
			if err := createCaddyPolicyHistory(tx, caddymodel.LegacyServerID, caddymodel.CaddyHistoryActionSimpleWafLastGood, previousConfig, previousModules); err != nil {
				return err
			}
		}

		if err := caddymodel.SaveConfig(tx, config, modules); err != nil {
			return fmt.Errorf("保存 Caddy 配置失败: %w", err)
		}

		history := &caddymodel.CaddyConfigHistory{
			ServerID: caddymodel.LegacyServerID,
			Action:   opts.HistoryAction,
			Hash:     hashConfig(config),
			Config:   config,
			Modules:  modules,
		}
		if err := tx.Create(history).Error; err != nil {
			return fmt.Errorf("创建 Caddy 配置历史失败: %w", err)
		}
		historyID = history.ID
		return nil
	}); err != nil {
		return 0, err
	}
	return historyID, nil
}

// mergeManagedBlocks 把库里保存的全局选项、:80 入口块模板与日志 snippet 定义合并进配置。
//
// 读库失败按"未配置"处理不阻断应用；模板本身不合法（缺 :80 地址、块头不对）属于
// 数据错误，直接报错避免把坏配置推给 Caddy。
//
// geo_auth 是唯一跟随开关的 snippet：IP 区域访问控制关闭时摘掉 import，避免留下
// 无意义的 forward_auth；开启时才补定义，保证引用必有定义又不污染配置。
func (s *caddyConfigApplyService) mergeManagedBlocks(config string) (string, error) {
	entryBlock := loadCaddyEntryBlock(s.svcCtx.DB)
	ipRegionEnabled := isIPRegionEnabled(s.svcCtx.DB)
	if !ipRegionEnabled {
		// IP 区域访问控制关闭时摘掉 import，避免留下无意义的 forward_auth
		entryBlock = removeCaddyDirectiveLine(entryBlock, "import geo_auth")
	}
	merged, err := mergeCaddyManagedBlocks(
		config,
		loadCaddyGlobalOptions(s.svcCtx.DB),
		entryBlock,
		loadCaddyLogSnippet(s.svcCtx.DB),
	)
	if err != nil {
		return "", err
	}
	if ipRegionEnabled {
		merged, _, err = ensureSnippetDefinition(merged, "geo_auth", renderGeoAuthSnippet())
		if err != nil {
			return "", err
		}
	}
	return merged, nil
}

// removeCaddyDirectiveLine 删掉配置中所有 trim 后等于该指令的行（含行尾换行）。
func removeCaddyDirectiveLine(config, directive string) string {
	lines := strings.Split(config, "\n")
	kept := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == directive {
			continue
		}
		kept = append(kept, line)
	}
	return strings.Join(kept, "\n")
}

// renderGeoAuthSnippet IP 区域访问控制的统一片段定义。
// forward_auth 直连后端 /api/internal/geo-check，403 直接透传给客户端。
func renderGeoAuthSnippet() string {
	return strings.Join([]string{
		"(geo_auth) {",
		"  forward_auth localhost:8888 {",
		"    uri /api/internal/geo-check",
		"    header_up X-Forwarded-Host {host}",
		"    header_up X-Forwarded-Uri {uri}",
		"    header_up X-Forwarded-Method {method}",
		"  }",
		"}",
	}, "\n")
}

// mergeCaddyManagedBlocks 是合并的纯函数形态：先合并全局选项，再替换入口块，
// 最后按最终位置插入或替换日志 snippet 定义。
//
// 日志 snippet 必须排在全局块之后：全局块交由 mergeGlobalOptions 原位重建，
// 先插 snippet 会打乱块头定位。
func mergeCaddyManagedBlocks(config string, opts CaddyGlobalOptionsValue, entryBlock, logSnippet string) (string, error) {
	merged := mergeGlobalOptions(config, opts)
	merged, err := mergeEntryBlock(merged, entryBlock)
	if err != nil {
		return "", err
	}
	merged, err = mergeCaddyLogSnippet(merged, logSnippet)
	if err != nil {
		return "", err
	}
	// waf_protect 与 subdomain-log 同层保证：入口块模板会 import 它，
	// 少了这一步 Caddy 会把 import 当文件 glob 解析并报 snippet not found。
	//
	// 仅在确实应用了入口块模板时补定义：空模板表示不接管入口块，
	// 此时配置里没有本层引入的 import，也就不该往配置里塞东西。
	if strings.TrimSpace(entryBlock) != "" {
		merged, _, err = ensureSnippetDefinition(merged, "waf_protect", renderWafProtectSnippet(detectCaddyNewline(merged)))
		if err != nil {
			return "", err
		}
	}
	// snippet 定义就位后，把存量站点侧旧 import 写法迁移为三参数单行
	return migrateSiteLogImports(merged), nil
}

func (s *caddyConfigApplyService) scheduleLogSourceSync(server *caddymodel.CaddyServer) {
	safego.New(context.Background(), "应用 Caddy 配置后同步日志源").Go(func() {
		// 先保证站点日志目录可被采集，再按配置里声明的路径补充精确源：
		// 目录源覆盖后续新建的站点文件，精确源覆盖显式配置的单文件路径。
		ingest.EnsureSiteLogSource(s.svcCtx.DB, s.logger)
		syncCaddyLogSources(s.svcCtx, server, s.logger)
	})
}

// findPreferredCaddyServer 返回唯一的 Caddy 管理目标（Admin API 地址与 Token）。
//
// 单实例下不存在「选哪个节点」，一律优先 local 行、其次 id 最小的行；
// 配置本身不在这张表里，这里只取连接信息。
func findPreferredCaddyServer(db *gorm.DB) (*caddymodel.CaddyServer, error) {
	if db == nil {
		return nil, fmt.Errorf("数据库为空")
	}

	var server caddymodel.CaddyServer
	err := db.Where("type = ?", "local").Order("id asc").First(&server).Error
	if err == gorm.ErrRecordNotFound {
		err = db.Order("id asc").First(&server).Error
	}
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("Caddy 服务器不存在")
		}
		return nil, fmt.Errorf("查询 Caddy 服务器失败: %w", err)
	}
	return &server, nil
}
