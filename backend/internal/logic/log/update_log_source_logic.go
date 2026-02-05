package log

import (
	"context"
	"fmt"
	"strings"
	"time"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/model"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type UpdateLogSourceLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateLogSourceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateLogSourceLogic {
	return &UpdateLogSourceLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateLogSourceLogic) UpdateLogSource(req *types.LogSourceUpdateReq) (resp *types.BaseResp, err error) {
	var source model.LogSource
	if err := l.svcCtx.DB.First(&source, req.ID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("log source not found")
		}
		return nil, err
	}

	oldPath := source.Path
	oldEnabled := source.Enabled

	if strings.TrimSpace(req.Name) != "" {
		source.Name = strings.TrimSpace(req.Name)
	}
	if req.Path != "" {
		path := strings.TrimSpace(req.Path)
		if path == "" {
			return nil, errInvalidLogSourcePath
		}
		source.Path = path
	}
	source.Enabled = req.Enabled
	source.UpdatedAt = time.Now()

	if err := l.svcCtx.DB.Save(&source).Error; err != nil {
		return nil, err
	}

	if oldEnabled && (source.Path != oldPath || !source.Enabled) && oldPath != "" {
		l.svcCtx.Ingestor.Stop(oldPath)
	}
	if source.Enabled && source.Path != "" {
		l.svcCtx.Ingestor.Start(source.Path)
	}

	return &types.BaseResp{
		Code: 200,
		Msg:  "success",
	}, nil
}
