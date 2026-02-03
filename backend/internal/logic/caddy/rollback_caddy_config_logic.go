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

type RollbackCaddyConfigLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRollbackCaddyConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RollbackCaddyConfigLogic {
	return &RollbackCaddyConfigLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RollbackCaddyConfigLogic) RollbackCaddyConfig(req *types.CaddyConfigRollbackReq) (resp *types.BaseResp, err error) {
	var server model.CaddyServer
	if err := l.svcCtx.DB.First(&server, req.ServerId).Error; err != nil {
		return nil, fmt.Errorf("server not found")
	}

	var history model.CaddyConfigHistory
	if err := l.svcCtx.DB.Where("id = ? AND server_id = ?", req.HistoryId, req.ServerId).First(&history).Error; err != nil {
		return nil, fmt.Errorf("history not found")
	}

	if err := adaptCaddyfile(&server, history.Config); err != nil {
		return nil, fmt.Errorf("caddy adapt failed: %v", err)
	}
	if err := loadCaddyfile(&server, history.Config); err != nil {
		return nil, fmt.Errorf("caddy api error: %v", err)
	}

	if err := l.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		server.Config = history.Config
		server.Modules = history.Modules
		if err := tx.Save(&server).Error; err != nil {
			return err
		}
		record := &model.CaddyConfigHistory{
			ServerID: server.ID,
			Action:   "rollback",
			Hash:     hashConfig(history.Config),
			Config:   history.Config,
			Modules:  history.Modules,
		}
		return tx.Create(record).Error
	}); err != nil {
		return nil, fmt.Errorf("failed to save config to database")
	}

	go syncCaddyLogSources(l.svcCtx, &server, l.Logger)

	return &types.BaseResp{
		Code: 200,
		Msg:  "success",
	}, nil
}
