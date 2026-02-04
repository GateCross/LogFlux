package caddy

import (
	"context"
	"fmt"
	"strings"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/model"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

const emptyModulesJSON = "{}"

type UpdateCaddyConfigLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateCaddyConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateCaddyConfigLogic {
	return &UpdateCaddyConfigLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateCaddyConfigLogic) UpdateCaddyConfig(req *types.CaddyConfigUpdateReq) (resp *types.BaseResp, err error) {
	var server model.CaddyServer
	if err := l.svcCtx.DB.First(&server, req.ServerId).Error; err != nil {
		return nil, fmt.Errorf("server not found")
	}

	modulesPayload := strings.TrimSpace(req.Modules)
	if modulesPayload == "" {
		if err := l.updateRawConfig(&server, req.Config); err != nil {
			return nil, err
		}
	} else {
		if err := l.updateStructuredConfig(&server, req.Config, req.Modules); err != nil {
			return nil, err
		}
	}

	// 4. 同步日志路径 (自动发现)
	go syncCaddyLogSources(l.svcCtx, &server, l.Logger)

	l.Logger.Info("Caddy config updated successfully")
	return &types.BaseResp{
		Code: 200,
		Msg:  "success",
	}, nil
}

func (l *UpdateCaddyConfigLogic) updateRawConfig(server *model.CaddyServer, config string) error {
	if err := l.pushCaddyConfig(server, config); err != nil {
		return err
	}
	modules := strings.TrimSpace(server.Modules)
	if modules == "" {
		modules = emptyModulesJSON
	}
	return l.persistCaddyConfig(server, config, modules)
}

func (l *UpdateCaddyConfigLogic) updateStructuredConfig(server *model.CaddyServer, config, modules string) error {
	if err := l.pushCaddyConfig(server, config); err != nil {
		return err
	}
	return l.persistCaddyConfig(server, config, modules)
}

func (l *UpdateCaddyConfigLogic) pushCaddyConfig(server *model.CaddyServer, config string) error {
	// 1. Pre-validate with /adapt
	if err := adaptCaddyfile(server, config); err != nil {
		l.Logger.Errorf("Caddy adapt failed: %v", err)
		return fmt.Errorf("caddy adapt failed: %v", err)
	}

	// 2. Push to Caddy API
	// Use /load endpoint with Content-Type: text/caddyfile
	// Caddy will compile it to JSON on the fly.
	l.Logger.Infof("Pushing Caddyfile to server %s (ID: %d)", server.Name, server.ID)
	if err := loadCaddyfile(server, config); err != nil {
		l.Logger.Errorf("Caddy load failed: %v", err)
		return fmt.Errorf("caddy api error: %v", err)
	}
	return nil
}

func (l *UpdateCaddyConfigLogic) persistCaddyConfig(server *model.CaddyServer, config, modules string) error {
	if err := l.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		server.Config = config
		server.Modules = modules
		if err := tx.Save(server).Error; err != nil {
			return err
		}
		history := &model.CaddyConfigHistory{
			ServerID: server.ID,
			Action:   "update",
			Hash:     hashConfig(config),
			Config:   config,
			Modules:  modules,
		}
		return tx.Create(history).Error
	}); err != nil {
		l.Logger.Errorf("Failed to save config history: %v", err)
		return fmt.Errorf("failed to save config to database")
	}
	return nil
}
