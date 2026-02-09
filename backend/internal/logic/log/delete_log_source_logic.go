package log

import (
	"context"
	"fmt"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/model"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
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
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("log source not found")
		}
		return nil, err
	}

	if err := l.svcCtx.DB.Delete(&model.LogSource{}, req.ID).Error; err != nil {
		return nil, err
	}

	if source.Path != "" {
		l.svcCtx.Ingestor.Stop(source.Path, source.Type)
	}

	return &types.BaseResp{
		Code: 200,
		Msg:  "success",
	}, nil
}
