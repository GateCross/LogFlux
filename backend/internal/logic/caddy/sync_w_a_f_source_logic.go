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

type SyncWAFSourceLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSyncWAFSourceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SyncWAFSourceLogic {
	return &SyncWAFSourceLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SyncWAFSourceLogic) SyncWAFSource(req *types.WAFSourceSyncReq) (resp *types.BaseResp, err error) {
	helper := newWAFLogicHelper(l.ctx, l.svcCtx, l.Logger)

	if err := helper.ensureStoreDirs(); err != nil {
		return nil, err
	}

	var source model.WAFSource
	if err := helper.svcCtx.DB.First(&source, req.ID).Error; err != nil {
		return nil, fmt.Errorf("source not found")
	}
	if !source.Enabled {
		return nil, fmt.Errorf("source is disabled")
	}
	if source.Mode != wafModeRemote {
		return nil, fmt.Errorf("source mode is not remote")
	}
	if strings.TrimSpace(source.URL) == "" {
		return nil, fmt.Errorf("source url is empty")
	}

	job := helper.startJob(source.ID, 0, "download", "manual")

	ext := detectPackageExt(source.URL)
	if ext == "" {
		ext = ".tar.gz"
	}
	tempName := fmt.Sprintf("%s_%d%s", sanitizeToken(source.Name), time.Now().UnixNano(), ext)
	tempPath := helper.store.TempPath(tempName)
	defer func() {
		if strings.TrimSpace(tempPath) != "" {
			_ = os.Remove(tempPath)
		}
	}()

	fetchResult, err := waf.FetchPackage(source.URL, tempPath, waf.FetchOptions{
		AllowedDomains: helper.svcCtx.Config.WAF.AllowedDomains,
		AuthType:       source.AuthType,
		AuthSecret:     source.AuthSecret,
		TimeoutSec:     60,
	})
	if err != nil {
		helper.updateSourceLastCheck(source.ID, "", err.Error())
		helper.finishJob(job, wafJobStatusFailed, err.Error(), 0)
		return nil, err
	}

	verifyResult, err := waf.VerifyPackage(fetchResult.SavedPath, waf.VerifyOptions{
		AllowedExt:      []string{".tar.gz", ".zip"},
		MaxPackageBytes: helper.svcCtx.Config.WAF.MaxPackageBytes,
	})
	if err != nil {
		helper.updateSourceLastCheck(source.ID, "", err.Error())
		helper.finishJob(job, wafJobStatusFailed, err.Error(), 0)
		return nil, err
	}

	version := ensureUniqueReleaseVersion(helper.svcCtx.DB, source.ID, deriveVersionFromURL(source.URL))
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
		MaxFiles:      helper.svcCtx.Config.WAF.ExtractMaxFiles,
		MaxTotalBytes: helper.svcCtx.Config.WAF.ExtractMaxTotalBytes,
	}); err != nil {
		_ = os.RemoveAll(releaseDir)
		helper.updateSourceLastCheck(source.ID, "", err.Error())
		helper.finishJob(job, wafJobStatusFailed, err.Error(), 0)
		return nil, err
	}

	release := &model.WAFRelease{
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

	if req.ActivateNow || source.AutoActivate {
		activateLogic := NewActivateWAFReleaseLogic(l.ctx, l.svcCtx)
		if _, activateErr := activateLogic.ActivateWAFRelease(&types.WAFReleaseActivateReq{ID: release.ID}); activateErr != nil {
			return nil, activateErr
		}
	}

	return &types.BaseResp{Code: 200, Msg: "success"}, nil
}
