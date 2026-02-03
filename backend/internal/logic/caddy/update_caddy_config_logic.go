package caddy

import (
	"context"
	"fmt"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/model"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

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

	// 1. Pre-validate with /adapt
	if err := adaptCaddyfile(&server, req.Config); err != nil {
		l.Logger.Errorf("Caddy adapt failed: %v", err)
		return nil, fmt.Errorf("caddy adapt failed: %v", err)
	}

	// 2. Push to Caddy API
	// Use /load endpoint with Content-Type: text/caddyfile
	// Caddy will compile it to JSON on the fly.

	l.Logger.Infof("Pushing Caddyfile to server %s (ID: %d)", server.Name, server.ID)

	if err := loadCaddyfile(&server, req.Config); err != nil {
		l.Logger.Errorf("Caddy load failed: %v", err)
		return nil, fmt.Errorf("caddy api error: %v", err)
	}

	// 3. Save to DB after successful push + record history
	newModules := server.Modules
	if req.Modules != "" {
		newModules = req.Modules
	}
	if err := l.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		server.Config = req.Config
		server.Modules = newModules
		if err := tx.Save(&server).Error; err != nil {
			return err
		}
		history := &model.CaddyConfigHistory{
			ServerID: server.ID,
			Action:   "update",
			Hash:     hashConfig(req.Config),
			Config:   req.Config,
			Modules:  newModules,
		}
		return tx.Create(history).Error
	}); err != nil {
		l.Logger.Errorf("Failed to save config history: %v", err)
		return nil, fmt.Errorf("failed to save config to database")
	}

	// 4. 同步日志路径 (自动发现)
	go syncCaddyLogSources(l.svcCtx, &server, l.Logger)

	l.Logger.Info("Caddy config updated successfully")
	return &types.BaseResp{
		Code: 200,
		Msg:  "success",
	}, nil
}
