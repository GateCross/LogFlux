package log

import (
	"context"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/model"

	"github.com/zeromicro/go-zero/core/logx"
)

type AddLogSourceLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewAddLogSourceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AddLogSourceLogic {
	return &AddLogSourceLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AddLogSourceLogic) AddLogSource(req *types.LogSourceReq) (resp *types.BaseResp, err error) {
	// Check if already exists
	var count int64
	l.svcCtx.DB.Model(&model.LogSource{}).Where("path = ?", req.Path).Count(&count)
	if count > 0 {
		return &types.BaseResp{Code: 1, Msg: "Source path already exists"}, nil
	}

	source := &model.LogSource{
		Name:    req.Name,
		Path:    req.Path,
		Type:    req.Type,
		Enabled: true, // Default enabled
	}

	if err := l.svcCtx.DB.Create(source).Error; err != nil {
		return nil, err
	}

	// Start monitoring immediately
	l.svcCtx.Ingestor.Start(source.Path)

	return &types.BaseResp{Code: 0, Msg: "Success"}, nil
}
