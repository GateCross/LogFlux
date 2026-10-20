package caddy

import (
	"context"
	"fmt"
	caddymodel "logflux/model/caddy"
	"strings"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/internal/utils/safego"

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
	var server caddymodel.CaddyServer
	if err := l.svcCtx.DB.WithContext(l.ctx).First(&server, req.ServerId).Error; err != nil {
		return nil, fmt.Errorf("服务器不存在")
	}

	var history caddymodel.CaddyConfigHistory
	if err := l.svcCtx.DB.WithContext(l.ctx).Where("id = ? AND server_id = ?", req.HistoryId, req.ServerId).First(&history).Error; err != nil {
		return nil, fmt.Errorf("历史记录不存在")
	}

	// 回滚前记录当前运行态，落库失败时用它恢复，避免运行态与库不一致。
	previousConfig := server.Config

	if err := adaptCaddyfile(&server, history.Config); err != nil {
		return nil, fmt.Errorf("Caddy 配置适配失败: %v", err)
	}
	if err := loadCaddyfile(&server, history.Config); err != nil {
		return nil, fmt.Errorf("Caddy API 错误: %v", err)
	}

	if err := l.svcCtx.DB.WithContext(l.ctx).Transaction(func(tx *gorm.DB) error {
		server.Config = history.Config
		server.Modules = history.Modules
		if err := tx.Save(&server).Error; err != nil {
			return err
		}
		record := &caddymodel.CaddyConfigHistory{
			ServerID: server.ID,
			Action:   caddymodel.CaddyHistoryActionRollback,
			Hash:     hashConfig(history.Config),
			Config:   history.Config,
			Modules:  history.Modules,
		}
		return tx.Create(record).Error
	}); err != nil {
		// 落库失败必须把运行态改回去，否则 Caddy 跑的是新配置、库里还是旧的。
		if strings.TrimSpace(previousConfig) != "" {
			if restoreErr := rollbackPolicyConfigToLastGood(&server, previousConfig); restoreErr != nil {
				return nil, fmt.Errorf("保存配置到数据库失败: %v，恢复原运行态失败: %v", err, restoreErr)
			}
		}
		return nil, fmt.Errorf("保存配置到数据库失败: %v", err)
	}

	safego.New(context.Background(), "回滚 Caddy 配置后同步日志源").Go(func() {
		syncCaddyLogSources(l.svcCtx, &server, l.Logger)
	})

	return &types.BaseResp{
		Code: 200,
		Msg:  "成功",
	}, nil
}
