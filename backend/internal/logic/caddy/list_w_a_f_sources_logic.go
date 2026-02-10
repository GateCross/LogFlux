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

type ListWAFSourcesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListWAFSourcesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListWAFSourcesLogic {
	return &ListWAFSourcesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListWAFSourcesLogic) ListWAFSources(req *types.WAFSourceListReq) (resp *types.WAFSourceListResp, err error) {
	helper := newWAFLogicHelper(l.ctx, l.svcCtx, l.Logger)

	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}

	db := helper.svcCtx.DB.Model(&model.WAFSource{})
	if kind := normalizeWAFKind(req.Kind); strings.TrimSpace(req.Kind) != "" {
		db = db.Where("kind = ?", kind)
	}
	if keyword := strings.TrimSpace(req.Name); keyword != "" {
		db = db.Where("name ILIKE ?", "%"+keyword+"%")
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("count sources failed: %w", err)
	}

	var sources []model.WAFSource
	offset := (page - 1) * pageSize
	if err := db.Order("updated_at desc, id desc").Limit(pageSize).Offset(offset).Find(&sources).Error; err != nil {
		return nil, fmt.Errorf("query sources failed: %w", err)
	}

	items := make([]types.WAFSourceItem, 0, len(sources))
	for _, source := range sources {
		items = append(items, types.WAFSourceItem{
			ID:           source.ID,
			Name:         source.Name,
			Kind:         source.Kind,
			Mode:         source.Mode,
			Url:          source.URL,
			ChecksumUrl:  source.ChecksumURL,
			AuthType:     source.AuthType,
			Schedule:     source.Schedule,
			Enabled:      source.Enabled,
			AutoCheck:    source.AutoCheck,
			AutoDownload: source.AutoDownload,
			AutoActivate: source.AutoActivate,
			LastRelease:  source.LastRelease,
			LastError:    source.LastError,
			CreatedAt:    formatTime(source.CreatedAt),
			UpdatedAt:    formatTime(source.UpdatedAt),
		})
	}

	return &types.WAFSourceListResp{List: items, Total: total}, nil
}
