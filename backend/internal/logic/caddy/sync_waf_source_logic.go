package caddy

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/internal/waf"
	"logflux/model"

	"github.com/zeromicro/go-zero/core/logx"
)

type SyncWafSourceLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSyncWafSourceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SyncWafSourceLogic {
	return &SyncWafSourceLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SyncWafSourceLogic) SyncWafSource(req *types.WafSourceSyncReq) (resp *types.BaseResp, err error) {
	helper := newWafLogicHelper(l.ctx, l.svcCtx, l.Logger)

	if err := helper.ensureStoreDirs(); err != nil {
		return nil, err
	}

	var source model.WafSource
	if err := helper.svcCtx.DB.First(&source, req.ID).Error; err != nil {
		return nil, fmt.Errorf("source not found")
	}
	if !source.Enabled {
		return nil, fmt.Errorf("source is disabled")
	}
	if normalizeWafKind(source.Kind) == wafKindCorazaEngine {
		return nil, fmt.Errorf("Coraza 引擎更新源无需手工同步，请直接使用引擎版本检查")
	}
	if source.Mode != wafModeRemote {
		return nil, fmt.Errorf("source mode is not remote")
	}
	if strings.TrimSpace(source.URL) == "" {
		return nil, fmt.Errorf("source url is empty")
	}

	fetchTimeoutSec := helper.svcCtx.Config.Waf.FetchTimeoutSec
	if fetchTimeoutSec <= 0 {
		fetchTimeoutSec = 180
	}

	downloadURL := strings.TrimSpace(source.URL)
	version := deriveVersionFromURL(downloadURL)
	if normalizeWafKind(source.Kind) == wafKindCRS {
		resolvedURL, resolvedVersion := helper.resolveCRSSyncTarget(&source)
		if strings.TrimSpace(resolvedURL) != "" {
			downloadURL = strings.TrimSpace(resolvedURL)
		}
		if strings.TrimSpace(resolvedVersion) != "" {
			version = strings.TrimSpace(resolvedVersion)
		}
	}

	job := helper.startJob(source.ID, 0, "download", "manual")
	existingRelease, err := findLatestReleaseByKindAndVersion(helper.svcCtx.DB, source.Kind, version)
	if err != nil {
		helper.updateSourceLastCheck(source.ID, version, err.Error())
		helper.finishJob(job, wafJobStatusFailed, err.Error(), 0)
		return nil, err
	}
	if existingRelease != nil && helper.canReuseRelease(existingRelease) {
		helper.updateSourceLastCheck(source.ID, existingRelease.Version, "")
		helper.finishJob(job, wafJobStatusSuccess, "版本已存在，复用已有版本", existingRelease.ID)

		if normalizeWafKind(source.Kind) != wafKindCorazaEngine && (req.ActivateNow || source.AutoActivate) {
			activateLogic := NewActivateWafReleaseLogic(l.ctx, l.svcCtx)
			if _, activateErr := activateLogic.ActivateWafRelease(&types.WafReleaseActivateReq{ID: existingRelease.ID}); activateErr != nil {
				return nil, activateErr
			}
		}

		return &types.BaseResp{Code: 200, Msg: "success"}, nil
	}

	ext := detectPackageExt(downloadURL)
	if ext == "" {
		ext = ".tar.gz"
	}
	tempName := fmt.Sprintf("%s_%d%s", sanitizeToken(source.Name), time.Now().UnixNano(), ext)
	tempPath := helper.store.StagePath(tempName)
	defer func() {
		if strings.TrimSpace(tempPath) != "" {
			_ = os.Remove(tempPath)
		}
	}()

	fetchResult, err := waf.FetchPackage(downloadURL, tempPath, waf.FetchOptions{
		AllowedDomains: helper.svcCtx.Config.Waf.AllowedDomains,
		AuthType:       source.AuthType,
		AuthSecret:     source.AuthSecret,
		ProxyURL:       source.ProxyURL,
		TimeoutSec:     fetchTimeoutSec,
	})
	if err != nil && strings.TrimSpace(source.ProxyURL) != "" {
		l.Logger.Errorf("proxy fetch failed, fallback direct connect: source=%s proxy=%s err=%v", source.Name, source.ProxyURL, err)
		fetchResult, err = waf.FetchPackage(downloadURL, tempPath, waf.FetchOptions{
			AllowedDomains: helper.svcCtx.Config.Waf.AllowedDomains,
			AuthType:       source.AuthType,
			AuthSecret:     source.AuthSecret,
			ProxyURL:       "",
			TimeoutSec:     fetchTimeoutSec,
		})
	}
	if err != nil {
		normalizedErr := normalizeWafSyncFetchError(err, strings.TrimSpace(source.ProxyURL) != "")
		l.Logger.Errorf("sync source fetch failed: source=%s url=%s timeoutSec=%d err=%v", source.Name, downloadURL, fetchTimeoutSec, err)
		helper.updateSourceLastCheck(source.ID, "", normalizedErr.Error())
		helper.finishJob(job, wafJobStatusFailed, normalizedErr.Error(), 0)
		return nil, normalizedErr
	}

	verifyResult, err := waf.VerifyPackage(fetchResult.SavedPath, waf.VerifyOptions{
		AllowedExt:      []string{".tar.gz", ".zip"},
		MaxPackageBytes: helper.svcCtx.Config.Waf.MaxPackageBytes,
	})
	if err != nil {
		helper.updateSourceLastCheck(source.ID, "", err.Error())
		helper.finishJob(job, wafJobStatusFailed, err.Error(), 0)
		return nil, err
	}

	packageName := fmt.Sprintf("%s_%s%s", sanitizeToken(source.Name), sanitizeToken(version), verifyResult.Ext)
	packagePath := helper.store.PackagePath(packageName)
	if err := os.Rename(fetchResult.SavedPath, packagePath); err != nil {
		helper.updateSourceLastCheck(source.ID, "", err.Error())
		helper.finishJob(job, wafJobStatusFailed, fmt.Sprintf("move package failed: %v", err), 0)
		return nil, fmt.Errorf("move package failed: %w", err)
	}
	tempPath = ""

	releaseDir := helper.store.ReleaseDir(version)
	if err := os.MkdirAll(releaseDir, 0o755); err != nil {
		helper.updateSourceLastCheck(source.ID, "", err.Error())
		helper.finishJob(job, wafJobStatusFailed, fmt.Sprintf("create release dir failed: %v", err), 0)
		return nil, fmt.Errorf("create release dir failed: %w", err)
	}

	if _, err := waf.ExtractPackage(packagePath, releaseDir, waf.ExtractOptions{
		MaxFiles:      helper.svcCtx.Config.Waf.ExtractMaxFiles,
		MaxTotalBytes: helper.svcCtx.Config.Waf.ExtractMaxTotalBytes,
	}); err != nil {
		_ = os.RemoveAll(releaseDir)
		helper.updateSourceLastCheck(source.ID, "", err.Error())
		helper.finishJob(job, wafJobStatusFailed, err.Error(), 0)
		return nil, err
	}

	release := &model.WafRelease{
		SourceID:     source.ID,
		Kind:         source.Kind,
		Version:      version,
		ArtifactType: artifactTypeFromExt(verifyResult.Ext),
		Checksum:     verifyResult.SHA256,
		SizeBytes:    verifyResult.SizeBytes,
		StoragePath:  filepath.Clean(releaseDir),
		Status:       wafReleaseStatusVerified,
	}

	if err := helper.svcCtx.DB.Create(release).Error; err != nil {
		helper.updateSourceLastCheck(source.ID, "", err.Error())
		helper.finishJob(job, wafJobStatusFailed, fmt.Sprintf("create release failed: %v", err), 0)
		return nil, fmt.Errorf("create release failed: %w", err)
	}

	helper.updateSourceLastCheck(source.ID, release.Version, "")
	helper.finishJob(job, wafJobStatusSuccess, "sync success", release.ID)

	if normalizeWafKind(source.Kind) != wafKindCorazaEngine && (req.ActivateNow || source.AutoActivate) {
		activateLogic := NewActivateWafReleaseLogic(l.ctx, l.svcCtx)
		if _, activateErr := activateLogic.ActivateWafRelease(&types.WafReleaseActivateReq{ID: release.ID}); activateErr != nil {
			return nil, activateErr
		}
	}

	return &types.BaseResp{Code: 200, Msg: "success"}, nil
}

func normalizeWafSyncFetchError(fetchErr error, hasProxy bool) error {
	if fetchErr == nil {
		return nil
	}

	raw := strings.ToLower(strings.TrimSpace(fetchErr.Error()))
	switch {
	case strings.Contains(raw, "context deadline exceeded"), strings.Contains(raw, "client.timeout exceeded"), strings.Contains(raw, "i/o timeout"):
		if hasProxy {
			return fmt.Errorf("下载源超时（代理与直连均失败），请检查代理连通性或稍后重试")
		}
		return fmt.Errorf("下载源超时，请配置可用代理后重试")
	case strings.Contains(raw, "host not allowed"):
		return fmt.Errorf("下载源域名未加入允许列表，请联系管理员在 Waf.AllowedDomains 中添加该域名")
	default:
		return fetchErr
	}
}
