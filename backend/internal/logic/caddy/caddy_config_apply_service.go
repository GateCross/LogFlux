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

	if strings.TrimSpace(previousConfig) == "" {
		historyID, err := s.persist(server, config, normalizedModules, action)
		if err != nil {
			server.Config = previousConfig
			server.Modules = previousModules
			return err
		}
		if err := loadCaddyfile(server, config); err != nil {
			if s != nil && s.logger != nil {
				s.logger.Errorf("Caddy 配置加载失败: %v", err)
			}
			if rollbackErr := s.restorePersisted(server, previousConfig, previousModules, historyID); rollbackErr != nil {
				return fmt.Errorf("Caddy 配置加载失败: %v，回滚数据库到未接管状态失败: %v", err, rollbackErr)
			}
			return fmt.Errorf("Caddy 配置加载失败，已回滚数据库到未接管状态: %w", err)
		}
		s.scheduleLogSourceSync(server)
		return nil
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

func (s *caddyConfigApplyService) restorePersisted(server *caddymodel.CaddyServer, config, modules string, historyID uint) error {
	if err := s.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&caddymodel.CaddyServer{}).
			Where("id = ?", server.ID).
			Updates(map[string]interface{}{
				"config":  config,
				"modules": modules,
			}).Error; err != nil {
			return fmt.Errorf("恢复 Caddy 服务器配置失败: %w", err)
		}
		if historyID == 0 {
			return nil
		}
		if err := tx.Where("id = ? AND server_id = ?", historyID, server.ID).Delete(&caddymodel.CaddyConfigHistory{}).Error; err != nil {
			return fmt.Errorf("删除未生效配置历史失败: %w", err)
		}
		return nil
	}); err != nil {
		return err
	}
	server.Config = config
	server.Modules = modules
	return nil
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
