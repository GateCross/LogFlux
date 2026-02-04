package caddy

import (
	"context"
	"fmt"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/model"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetCaddyConfigHistoryDetailLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetCaddyConfigHistoryDetailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetCaddyConfigHistoryDetailLogic {
	return &GetCaddyConfigHistoryDetailLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetCaddyConfigHistoryDetailLogic) GetCaddyConfigHistoryDetail(req *types.CaddyConfigHistoryDetailReq) (resp *types.CaddyConfigHistoryDetailResp, err error) {
	var history model.CaddyConfigHistory
	if err := l.svcCtx.DB.First(&history, "id = ? AND server_id = ?", req.HistoryId, req.ServerId).Error; err != nil {
		return nil, fmt.Errorf("history not found")
	}

	return &types.CaddyConfigHistoryDetailResp{
		ID:        history.ID,
		ServerId:  history.ServerID,
		Action:    history.Action,
		Hash:      history.Hash,
		Config:    history.Config,
		Modules:   history.Modules,
		CreatedAt: history.CreatedAt.Format("2006-01-02 15:04:05"),
	}, nil
}
