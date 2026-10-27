package caddy

import (
	"context"
	"fmt"

	caddymodel "logflux/model/caddy"

	"logflux/internal/svc"
	"logflux/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
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
	db := l.svcCtx.DB.WithContext(l.ctx)
	server, err := findPreferredCaddyServer(db)
	if err != nil {
		return nil, err
	}

	var history caddymodel.CaddyConfigHistory
	if err := db.Where("id = ?", req.HistoryId).First(&history).Error; err != nil {
		return nil, fmt.Errorf("历史记录不存在")
	}

	applyService := newCaddyConfigApplyService(l.svcCtx, l.Logger)
	if err := applyService.Apply(server, history.Config, history.Modules, configWriteOptions{
		HistoryAction: caddymodel.CaddyHistoryActionRollback,
		BaseHash:      req.BaseHash,
	}); err != nil {
		return nil, err
	}

	return &types.BaseResp{
		Code: 200,
		Msg:  "成功",
	}, nil
}
