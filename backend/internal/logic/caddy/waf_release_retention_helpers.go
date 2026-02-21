package caddy

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"logflux/model"

	"gorm.io/gorm"
)

func (helper *wafLogicHelper) applyReleaseRetention(kind string) {
	if helper == nil || helper.svcCtx == nil || helper.svcCtx.DB == nil {
		return
	}

	keepCount := helper.releaseRetentionCount()
	if keepCount <= 0 {
		return
	}

	if err := helper.pruneOldReleases(kind, keepCount); err != nil {
		helper.logger.Errorf("apply release retention failed: kind=%s keep=%d err=%v", kind, keepCount, err)
	}
}

func (helper *wafLogicHelper) releaseRetentionCount() int {
	keepCount := helper.svcCtx.Config.Waf.ReleaseRetentionCount
	if keepCount <= 0 {
		keepCount = 20
	}
	return keepCount
}

func (helper *wafLogicHelper) pruneOldReleases(kind string, keepCount int) error {
	if keepCount <= 0 {
		return nil
	}

	normalizedKind := normalizeWafKind(kind)
	var releases []model.WafRelease
	if err := helper.svcCtx.DB.
		Where("kind = ?", normalizedKind).
		Order("created_at desc, id desc").
		Find(&releases).Error; err != nil {
		return fmt.Errorf("query release retention candidates failed: %w", err)
	}
	if len(releases) <= keepCount {
		return nil
	}

	pinnedPathSet := helper.collectPinnedReleasePathSet()
	keepIDSet := make(map[uint]struct{}, keepCount)
	for i, release := range releases {
		if i < keepCount || release.Status == wafReleaseStatusActive || pinnedPathSet[filepath.Clean(strings.TrimSpace(release.StoragePath))] {
			keepIDSet[release.ID] = struct{}{}
		}
	}
	if len(keepIDSet) >= len(releases) {
		return nil
	}

	keptPathSet := make(map[string]struct{}, len(keepIDSet))
	for _, release := range releases {
		if _, keep := keepIDSet[release.ID]; !keep {
			continue
		}
		safePath, pathErr := helper.ensurePathInWorkDir(release.StoragePath)
		if pathErr != nil {
			continue
		}
		keptPathSet[filepath.Clean(safePath)] = struct{}{}
	}

	deleteIDs := make([]uint, 0, len(releases)-len(keepIDSet))
	pathsToRemove := make([]string, 0, len(releases)-len(keepIDSet))
	for _, release := range releases {
		if _, keep := keepIDSet[release.ID]; keep {
			continue
		}

		deleteIDs = append(deleteIDs, release.ID)
		safePath, pathErr := helper.ensurePathInWorkDir(release.StoragePath)
		if pathErr != nil {
			continue
		}
		cleanPath := filepath.Clean(safePath)
		if _, usedByKept := keptPathSet[cleanPath]; usedByKept {
			continue
		}
		pathsToRemove = append(pathsToRemove, cleanPath)
	}
	if len(deleteIDs) == 0 {
		return nil
	}

	if err := helper.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("release_id IN ?", deleteIDs).Delete(&model.WafUpdateJob{}).Error; err != nil {
			return fmt.Errorf("delete retained release jobs failed: %w", err)
		}
		if err := tx.Where("id IN ?", deleteIDs).Delete(&model.WafRelease{}).Error; err != nil {
			return fmt.Errorf("delete retained releases failed: %w", err)
		}
		return nil
	}); err != nil {
		return err
	}

	for _, pathValue := range dedupeNonEmptyStrings(pathsToRemove) {
		if removeErr := os.RemoveAll(pathValue); removeErr != nil {
			helper.logger.Errorf("remove retained release path failed: path=%s err=%v", pathValue, removeErr)
		}
	}
	return nil
}

func (helper *wafLogicHelper) collectPinnedReleasePathSet() map[string]bool {
	pinnedPathSet := make(map[string]bool, 2)
	if helper == nil || helper.store == nil {
		return pinnedPathSet
	}

	linkPaths := []string{
		helper.store.CurrentLinkPath(),
		helper.store.LastGoodLinkPath(),
	}
	for _, linkPath := range linkPaths {
		targetPath, err := helper.store.LinkTarget(linkPath)
		if err != nil {
			continue
		}
		safePath, pathErr := helper.ensurePathInWorkDir(targetPath)
		if pathErr != nil {
			continue
		}
		pinnedPathSet[filepath.Clean(strings.TrimSpace(safePath))] = true
	}
	return pinnedPathSet
}

func dedupeNonEmptyStrings(items []string) []string {
	if len(items) == 0 {
		return nil
	}

	set := make(map[string]struct{}, len(items))
	results := make([]string, 0, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		if _, exists := set[trimmed]; exists {
			continue
		}
		set[trimmed] = struct{}{}
		results = append(results, trimmed)
	}
	return results
}
