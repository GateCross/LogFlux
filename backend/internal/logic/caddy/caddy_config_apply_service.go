package caddy

import (
	"fmt"
	"os"
	"strings"

	"logflux/internal/svc"
	"logflux/model"

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

func (s *caddyConfigApplyService) loadCurrent(server *model.CaddyServer) (string, string, error) {
	if server == nil {
		return "", emptyModulesJSON, fmt.Errorf("caddy server not found")
	}

	if trimmed := strings.TrimSpace(server.Config); trimmed != "" {
		return server.Config, normalizeCaddyModulesJSON(server.Modules), nil
	}

	if strings.EqualFold(server.Type, "local") {
		raw, err := os.ReadFile("/etc/caddy/Caddyfile")
		if err == nil {
			config := strings.TrimSpace(string(raw))
			if config != "" {
				return config, normalizeCaddyModulesJSON(server.Modules), nil
			}
		}
	}

	return "", normalizeCaddyModulesJSON(server.Modules), fmt.Errorf("caddy config is empty, please save caddy config first")
}

func (s *caddyConfigApplyService) apply(server *model.CaddyServer, config, modules, action string) error {
	if server == nil {
		return fmt.Errorf("caddy server not found")
	}

	normalizedModules := normalizeCaddyModulesJSON(modules)
	if err := adaptCaddyfile(server, config); err != nil {
		if s != nil && s.logger != nil {
			s.logger.Errorf("Caddy adapt failed: %v", err)
		}
		return err
	}
	if err := loadCaddyfile(server, config); err != nil {
		if s != nil && s.logger != nil {
			s.logger.Errorf("Caddy load failed: %v", err)
		}
		return err
	}

	if err := s.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		server.Config = config
		server.Modules = normalizedModules
		if err := tx.Save(server).Error; err != nil {
			return fmt.Errorf("save caddy server config failed: %w", err)
		}

		history := &model.CaddyConfigHistory{
			ServerID: server.ID,
			Action:   strings.TrimSpace(action),
			Hash:     hashConfig(config),
			Config:   config,
			Modules:  normalizedModules,
		}
		if err := tx.Create(history).Error; err != nil {
			return fmt.Errorf("create caddy config history failed: %w", err)
		}
		return nil
	}); err != nil {
		return err
	}

	go syncCaddyLogSources(s.svcCtx, server, s.logger)
	return nil
}

func findPreferredCaddyServer(db *gorm.DB, serverID uint) (*model.CaddyServer, error) {
	if db == nil {
		return nil, fmt.Errorf("db is nil")
	}

	var server model.CaddyServer
	if serverID > 0 {
		if err := db.First(&server, serverID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil, fmt.Errorf("caddy server not found")
			}
			return nil, fmt.Errorf("query caddy server failed: %w", err)
		}
		return &server, nil
	}

	err := db.Where("type = ?", "local").Order("id asc").First(&server).Error
	if err == gorm.ErrRecordNotFound {
		err = db.Order("id asc").First(&server).Error
	}
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("caddy server not found")
		}
		return nil, fmt.Errorf("query caddy server failed: %w", err)
	}
	return &server, nil
}
