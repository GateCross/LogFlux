package caddy

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/model"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type ClearWafReleasesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewClearWafReleasesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ClearWafReleasesLogic {
	return &ClearWafReleasesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ClearWafReleasesLogic) ClearWafReleases(req *types.WafReleaseClearReq) (resp *types.BaseResp, err error) {
	helper := newWafLogicHelper(l.ctx, l.svcCtx, l.Logger)
	kind := normalizeWafKind(req.Kind)
	if kind == "" {
		kind = wafKindCRS
	}
	if kind != wafKindCRS {
		return nil, fmt.Errorf("仅支持清空 CRS 非激活版本")
	}

	var candidates []model.WafRelease
	if err := l.svcCtx.DB.
		Where("kind = ? AND status <> ?", kind, wafReleaseStatusActive).
		Order("id asc").
		Find(&candidates).Error; err != nil {
		return nil, fmt.Errorf("query clear candidates failed: %w", err)
	}

	if len(candidates) == 0 {
		return &types.BaseResp{Code: 200, Msg: "success"}, nil
	}

	activePathSet := make(map[string]struct{})
	var activePaths []string
	if err := l.svcCtx.DB.Model(&model.WafRelease{}).
		Where("kind = ? AND status = ?", kind, wafReleaseStatusActive).
		Pluck("storage_path", &activePaths).Error; err != nil {
		return nil, fmt.Errorf("query active release paths failed: %w", err)
	}
	for _, activePath := range activePaths {
		safePath, pathErr := helper.ensurePathInWorkDir(activePath)
		if pathErr != nil {
			continue
		}
		activePathSet[filepath.Clean(safePath)] = struct{}{}
	}

	releaseIDs := make([]uint, 0, len(candidates))
	pathsToRemove := make([]string, 0, len(candidates))
	for _, item := range candidates {
		releaseIDs = append(releaseIDs, item.ID)

		storagePath := strings.TrimSpace(item.StoragePath)
		if storagePath == "" {
			continue
		}
		safePath, pathErr := helper.ensurePathInWorkDir(storagePath)
		if pathErr != nil {
			l.Logger.Errorf("skip unsafe release path: id=%d path=%s err=%v", item.ID, storagePath, pathErr)
			continue
		}
		cleanPath := filepath.Clean(safePath)
		if _, keep := activePathSet[cleanPath]; keep {
			continue
		}
		pathsToRemove = append(pathsToRemove, cleanPath)
	}

	if err := l.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		if len(releaseIDs) > 0 {
			if err := tx.Where("release_id IN ?", releaseIDs).Delete(&model.WafUpdateJob{}).Error; err != nil {
				return fmt.Errorf("delete related waf jobs failed: %w", err)
			}
			if err := tx.Where("id IN ?", releaseIDs).Delete(&model.WafRelease{}).Error; err != nil {
				return fmt.Errorf("delete waf releases failed: %w", err)
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}

	for _, pathValue := range dedupeNonEmptyStrings(pathsToRemove) {
		if removeErr := os.RemoveAll(pathValue); removeErr != nil {
			l.Logger.Errorf("remove release storage path failed: path=%s err=%v", pathValue, removeErr)
		}
	}

	return &types.BaseResp{Code: 200, Msg: "success"}, nil
}
