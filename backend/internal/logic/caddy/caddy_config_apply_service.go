package caddy

import (
	"context"
	"fmt"
	caddymodel "logflux/model/caddy"
	"strings"
	"sync"

	"logflux/internal/svc"
	"logflux/internal/utils/safego"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

// caddyConfigWriteMu 串行化所有 Caddy 配置写路径。
//
// adapt → load → persist 必须作为整体执行：若无互斥，两个并发写请求交错
// （A.load → B.load → B.persist → A.persist）会出现运行态与库内配置分叉，
// 且双方都返回成功；配置写频率极低，全局单锁即可，无需按服务器细分。
var caddyConfigWriteMu sync.Mutex

type caddyConfigApplyService struct {
	svcCtx *svc.ServiceContext
	logger logx.Logger
}

func newCaddyConfigApplyService(svcCtx *svc.ServiceContext, logger logx.Logger) *caddyConfigApplyService {
	return &caddyConfigApplyService{svcCtx: svcCtx, logger: logger}
}

func (s *caddyConfigApplyService) loadCurrent(server *caddymodel.CaddyServer) (string, string, error) {
	return loadCurrentCaddyConfig(server)
}

// checkBaseHash 做乐观并发检查。
//
// baseHash 为空表示旧版客户端，保持向后兼容直接放行；非空时与库内当前哈希比对，
// 不一致说明期间有人改过配置，继续写会静默覆盖对方的改动。
func checkBaseHash(server *caddymodel.CaddyServer, baseHash string) error {
	baseHash = strings.TrimSpace(baseHash)
	if baseHash == "" || server == nil {
		return nil
	}
	current := hashConfig(server.Config)
	if current != baseHash {
		return fmt.Errorf("配置已被他人修改，请刷新后重试")
	}
	return nil
}

// verifyBaseHash 从库里重新取一份最新配置再校验，避免用请求上下文里的旧快照判断。
func verifyBaseHash(db *gorm.DB, serverID uint, baseHash string) error {
	if strings.TrimSpace(baseHash) == "" {
		return nil
	}
	// serverID 为 0 表示"默认服务器"，必须先解析成实际记录再比对。
	server, err := findPreferredCaddyServer(db, serverID)
	if err != nil {
		return err
	}
	return checkBaseHash(server, baseHash)
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

	// 锁内重读库内最新行再校验 baseHash：请求开始时的内存快照不可信，
	// 从加载 server 到落库之间隔着 adapt/load 两次带重试的 HTTP 调用（窗口可达数十秒），
	// 期间他人落库的修改若不在此重新比对，会被携带"当时合法"哈希的后写者覆盖。
	var fresh caddymodel.CaddyServer
	if err := s.svcCtx.DB.First(&fresh, server.ID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("Caddy 服务器不存在")
		}
		return fmt.Errorf("查询 Caddy 服务器失败: %w", err)
	}
	if err := checkBaseHash(&fresh, opts.BaseHash); err != nil {
		return err
	}
	// 以库内最新值为回滚基准，并同步回调用方持有的对象，落库与响应展示都以它为准。
	*server = fresh

	normalizedModules := normalizeCaddyModulesJSON(modules)
	previousConfig := server.Config
	previousModules := server.Modules

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

	if _, err := s.persist(server, config, normalizedModules, opts, previousConfig, previousModules); err != nil {
		server.Config = previousConfig
		server.Modules = previousModules
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

func (s *caddyConfigApplyService) persist(server *caddymodel.CaddyServer, config, modules string, opts configWriteOptions, previousConfig, previousModules string) (uint, error) {
	var historyID uint
	if err := s.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		if opts.InTx != nil {
			if err := opts.InTx(tx); err != nil {
				return err
			}
		}
		if opts.SnapshotBefore {
			if err := createCaddyPolicyHistory(tx, server.ID, caddymodel.CaddyHistoryActionSimpleWafLastGood, previousConfig, previousModules); err != nil {
				return err
			}
		}

		server.Config = config
		server.Modules = modules
		if err := tx.Save(server).Error; err != nil {
			return fmt.Errorf("保存 Caddy 服务器配置失败: %w", err)
		}

		history := &caddymodel.CaddyConfigHistory{
			ServerID: server.ID,
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

func (s *caddyConfigApplyService) scheduleLogSourceSync(server *caddymodel.CaddyServer) {
	safego.New(context.Background(), "应用 Caddy 配置后同步日志源").Go(func() {
		syncCaddyLogSources(s.svcCtx, server, s.logger)
	})
}

func findPreferredCaddyServer(db *gorm.DB, serverID uint) (*caddymodel.CaddyServer, error) {
	if db == nil {
		return nil, fmt.Errorf("数据库为空")
	}

	var server caddymodel.CaddyServer
	if serverID > 0 {
		if err := db.First(&server, serverID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil, fmt.Errorf("Caddy 服务器不存在")
			}
			return nil, fmt.Errorf("查询 Caddy 服务器失败: %w", err)
		}
		return &server, nil
	}

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
