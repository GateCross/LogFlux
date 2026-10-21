package caddy

import (
	"context"
	"fmt"
	caddymodel "logflux/model/caddy"
	"strings"

	"logflux/internal/svc"
	"logflux/internal/utils/safego"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

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

// apply 按「校验 → 加载 → 落库」固定顺序应用配置。
//
// 顺序统一的原因：/load 本身失败会自动回滚运行态，落库前出错等于什么都没改；
// 只有落库失败才需要把运行态改回去，而那次恢复依赖上一份配置，所以先加载再落库
// 是唯一能把"哪一步失败、要回滚什么"说清楚的顺序。首次接管同样适用，此前它
// 反着来（先落库再加载），失败语义与其他路径不一致。
func (s *caddyConfigApplyService) apply(server *caddymodel.CaddyServer, config, modules, action string) error {
	if server == nil {
		return fmt.Errorf("Caddy 服务器不存在")
	}

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

	if _, err := s.persist(server, config, normalizedModules, action); err != nil {
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

func (s *caddyConfigApplyService) persist(server *caddymodel.CaddyServer, config, modules, action string) (uint, error) {
	var historyID uint
	if err := s.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		server.Config = config
		server.Modules = modules
		if err := tx.Save(server).Error; err != nil {
			return fmt.Errorf("保存 Caddy 服务器配置失败: %w", err)
		}

		history := &caddymodel.CaddyConfigHistory{
			ServerID: server.ID,
			Action:   strings.TrimSpace(action),
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
