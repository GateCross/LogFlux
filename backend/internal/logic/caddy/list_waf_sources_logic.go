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

type ListWafSourcesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListWafSourcesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListWafSourcesLogic {
	return &ListWafSourcesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListWafSourcesLogic) ListWafSources(req *types.WafSourceListReq) (resp *types.WafSourceListResp, err error) {
	helper := newWafLogicHelper(l.ctx, l.svcCtx, l.Logger)
	l.svcCtx.EnsureWafDefaultSources()

	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}

	db := helper.svcCtx.DB.Model(&model.WafSource{})
	rawKind := strings.TrimSpace(req.Kind)
	hasKindFilter := false
	if rawKind != "" {
		kind := normalizeWafKind(rawKind)
		if kind == wafKindCorazaEngine {
			return &types.WafSourceListResp{List: []types.WafSourceItem{}, Total: 0}, nil
		}
		if validateWafKind(kind) == nil {
			db = db.Where("kind = ?", kind)
			hasKindFilter = true
		} else {
			l.Logger.Infof("ignore invalid waf source kind filter: %s", rawKind)
		}
	}
	hasNameFilter := false
	if keyword := strings.TrimSpace(req.Name); keyword != "" {
		hasNameFilter = true
		db = db.Where("name ILIKE ?", "%"+keyword+"%")
	}
	db = db.Where("kind <> ?", wafKindCorazaEngine)

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("count sources failed: %w", err)
	}

	if total == 0 && !hasKindFilter && !hasNameFilter {
		l.svcCtx.EnsureWafDefaultSources()
		if err := db.Count(&total).Error; err != nil {
			return nil, fmt.Errorf("count sources after ensure defaults failed: %w", err)
		}
	}

	var sources []model.WafSource
	offset := (page - 1) * pageSize
	if err := db.Order("updated_at desc, id desc").Limit(pageSize).Offset(offset).Find(&sources).Error; err != nil {
		return nil, fmt.Errorf("query sources failed: %w", err)
	}

	items := make([]types.WafSourceItem, 0, len(sources))
	for _, source := range sources {
		items = append(items, types.WafSourceItem{
			ID:           source.ID,
			Name:         source.Name,
			Kind:         source.Kind,
			Mode:         source.Mode,
			Url:          source.URL,
			ChecksumUrl:  source.ChecksumURL,
			ProxyUrl:     source.ProxyURL,
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

	return &types.WafSourceListResp{List: items, Total: total}, nil
}
