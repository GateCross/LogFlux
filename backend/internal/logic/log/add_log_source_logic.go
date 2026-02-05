package log

import (
	"context"
	"strings"
	"time"

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
	name := strings.TrimSpace(req.Name)
	path := strings.TrimSpace(req.Path)
	sourceType := strings.TrimSpace(req.Type)
	if sourceType == "" {
		sourceType = "caddy"
	}
	if name == "" {
		name = path
	}
	if path == "" {
		return nil, errInvalidLogSourcePath
	}

	source := &model.LogSource{
		Name:      name,
		Path:      path,
		Type:      sourceType,
		Enabled:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := l.svcCtx.DB.Create(source).Error; err != nil {
		return nil, err
	}

	l.svcCtx.Ingestor.Start(source.Path)

	return &types.BaseResp{
		Code: 200,
		Msg:  "success",
	}, nil
}
