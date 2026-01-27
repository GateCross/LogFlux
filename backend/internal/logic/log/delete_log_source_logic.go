package log

import (
	"context"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/model"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteLogSourceLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteLogSourceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteLogSourceLogic {
	return &DeleteLogSourceLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteLogSourceLogic) DeleteLogSource(req *types.IDReq) (resp *types.BaseResp, err error) {
	var source model.LogSource
	if err := l.svcCtx.DB.First(&source, req.ID).Error; err != nil {
		return nil, err
	}

	// Stop monitoring
	l.svcCtx.Ingestor.Stop(source.Path)

	// Delete from DB
	if err := l.svcCtx.DB.Delete(&source).Error; err != nil {
		return nil, err
	}

	return &types.BaseResp{Code: 0, Msg: "Success"}, nil
}
