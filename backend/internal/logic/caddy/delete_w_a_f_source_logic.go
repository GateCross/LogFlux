package caddy

import (
	"context"
	"fmt"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/model"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteWAFSourceLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteWAFSourceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteWAFSourceLogic {
	return &DeleteWAFSourceLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteWAFSourceLogic) DeleteWAFSource(req *types.IDReq) (resp *types.BaseResp, err error) {
	helper := newWAFLogicHelper(l.ctx, l.svcCtx, l.Logger)

	if err := helper.svcCtx.DB.Delete(&model.WAFSource{}, req.ID).Error; err != nil {
		return nil, fmt.Errorf("delete source failed: %w", err)
	}

	return &types.BaseResp{Code: 200, Msg: "success"}, nil
}
