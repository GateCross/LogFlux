package caddy

import (
	"context"
	"fmt"
	"strings"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/model"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListWafReleasesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListWafReleasesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListWafReleasesLogic {
	return &ListWafReleasesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListWafReleasesLogic) ListWafReleases(req *types.WafReleaseListReq) (resp *types.WafReleaseListResp, err error) {
	helper := newWafLogicHelper(l.ctx, l.svcCtx, l.Logger)

	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}

	db := helper.svcCtx.DB.Model(&model.WafRelease{})
	if strings.TrimSpace(req.Kind) != "" {
		db = db.Where("kind = ?", normalizeWafKind(req.Kind))
	}
	if status := strings.TrimSpace(req.Status); status != "" {
		db = db.Where("status = ?", strings.ToLower(status))
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("count releases failed: %w", err)
	}

	var releases []model.WafRelease
	offset := (page - 1) * pageSize
	if err := db.Order("created_at desc, id desc").Limit(pageSize).Offset(offset).Find(&releases).Error; err != nil {
		return nil, fmt.Errorf("query releases failed: %w", err)
	}

	items := make([]types.WafReleaseItem, 0, len(releases))
	for _, release := range releases {
		items = append(items, types.WafReleaseItem{
			ID:           release.ID,
			SourceId:     release.SourceID,
			Kind:         release.Kind,
			Version:      release.Version,
			ArtifactType: release.ArtifactType,
			Checksum:     release.Checksum,
			SizeBytes:    release.SizeBytes,
			StoragePath:  release.StoragePath,
			Status:       release.Status,
			CreatedAt:    formatTime(release.CreatedAt),
			UpdatedAt:    formatTime(release.UpdatedAt),
		})
	}

	return &types.WafReleaseListResp{List: items, Total: total}, nil
}
