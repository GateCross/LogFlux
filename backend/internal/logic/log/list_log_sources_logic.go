package log

import (
	"context"
	"fmt"

	"logflux/internal/ingest"
	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/model"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListLogSourcesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListLogSourcesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListLogSourcesLogic {
	return &ListLogSourcesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListLogSourcesLogic) ListLogSources(req *types.LogSourceListReq) (resp *types.LogSourceListResp, err error) {
	var (
		sources []model.LogSource
		total   int64
	)

	db := l.svcCtx.DB.Model(&model.LogSource{})
	if err := db.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("count log sources failed: %w", err)
	}

	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}
	offset := (req.Page - 1) * req.PageSize
	if err := db.Order("created_at desc, id desc").Limit(req.PageSize).Offset(offset).Find(&sources).Error; err != nil {
		return nil, err
	}

	list := make([]types.LogSourceItem, 0, len(sources))
	for _, source := range sources {
		scanInterval := source.ScanInterval
		if scanInterval <= 0 {
			scanInterval = ingest.DefaultScanIntervalSec()
		}
		list = append(list, types.LogSourceItem{
			ID:           source.ID,
			Name:         source.Name,
			Path:         source.Path,
			Type:         source.Type,
			Enabled:      source.Enabled,
			ScanInterval: scanInterval,
			CreatedAt:    source.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return &types.LogSourceListResp{
		List:  list,
		Total: total,
	}, nil
}
