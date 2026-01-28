package caddy

import (
	"context"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/model"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteCaddyServerLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteCaddyServerLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteCaddyServerLogic {
	return &DeleteCaddyServerLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteCaddyServerLogic) DeleteCaddyServer(req *types.IDReq) (resp *types.BaseResp, err error) {
	if err := l.svcCtx.DB.Delete(&model.CaddyServer{}, req.ID).Error; err != nil {
		return nil, err
	}

	return &types.BaseResp{
		Code: 200,
		Msg:  "success",
	}, nil
}
