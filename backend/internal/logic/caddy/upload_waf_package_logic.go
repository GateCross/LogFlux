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

type UploadWafPackageLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUploadWafPackageLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UploadWafPackageLogic {
	return &UploadWafPackageLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UploadWafPackageLogic) UploadWafPackage(req *types.WafUploadReq) (resp *types.BaseResp, err error) {
	helper := newWafLogicHelper(l.ctx, l.svcCtx, l.Logger)

	if err := helper.ensureStoreDirs(); err != nil {
		return nil, err
	}

	kind := normalizeWafKind(req.Kind)
	if err := validateWafKind(kind); err != nil {
		return nil, err
	}

	version := strings.TrimSpace(req.Version)
	if version == "" {
		return nil, fmt.Errorf("version is required")
	}
	version = sanitizeToken(version)

	tempPath, _ := l.ctx.Value(wafUploadTempPathCtxKey).(string)
	tempPath = strings.TrimSpace(tempPath)
	if tempPath == "" {
		return nil, fmt.Errorf("upload file is required")
	}

	safePath, safeErr := helper.ensurePathInWorkDir(tempPath)
	if safeErr != nil {
		return nil, safeErr
	}
	tempPath = safePath

	fileName, _ := l.ctx.Value(wafUploadFileNameCtxKey).(string)
	if strings.TrimSpace(fileName) == "" {
		fileName = basenameSafe(tempPath)
	}

	defer func() {
		if strings.TrimSpace(tempPath) != "" {
			_ = os.Remove(tempPath)
		}
	}()

	job := helper.startJob(0, 0, "verify", "upload")
	existingRelease, err := findLatestReleaseByKindAndVersion(helper.svcCtx.DB, kind, version)
	if err != nil {
		helper.finishJob(job, wafJobStatusFailed, err.Error(), 0)
		return nil, err
	}
	if existingRelease != nil && helper.canReuseRelease(existingRelease) {
		helper.finishJob(job, wafJobStatusSuccess, "版本已存在，复用已有版本", existingRelease.ID)
		if kind != wafKindCorazaEngine && req.ActivateNow {
			activateLogic := NewActivateWafReleaseLogic(l.ctx, l.svcCtx)
			if _, activateErr := activateLogic.ActivateWafRelease(&types.WafReleaseActivateReq{ID: existingRelease.ID}); activateErr != nil {
				return nil, activateErr
			}
		}
		return &types.BaseResp{Code: 200, Msg: "success"}, nil
	}

	verifyResult, err := waf.VerifyPackage(tempPath, waf.VerifyOptions{
		AllowedExt:      []string{".tar.gz", ".zip"},
		MaxPackageBytes: helper.svcCtx.Config.Waf.MaxPackageBytes,
		ExpectedSHA256:  strings.TrimSpace(req.Checksum),
	})
	if err != nil {
		helper.finishJob(job, wafJobStatusFailed, err.Error(), 0)
		return nil, err
	}

	packageName := fmt.Sprintf("upload_%s_%d%s", sanitizeToken(version), time.Now().UnixNano(), verifyResult.Ext)
	packagePath := helper.store.PackagePath(packageName)
	if err := os.Rename(tempPath, packagePath); err != nil {
		helper.finishJob(job, wafJobStatusFailed, fmt.Sprintf("move package failed: %v", err), 0)
		return nil, fmt.Errorf("move package failed: %w", err)
	}
	tempPath = ""

	releaseDir := helper.store.ReleaseDir(version)
	if err := os.MkdirAll(releaseDir, 0o755); err != nil {
		helper.finishJob(job, wafJobStatusFailed, fmt.Sprintf("create release dir failed: %v", err), 0)
		return nil, fmt.Errorf("create release dir failed: %w", err)
	}

	if _, err := waf.ExtractPackage(packagePath, releaseDir, waf.ExtractOptions{
		MaxFiles:      helper.svcCtx.Config.Waf.ExtractMaxFiles,
		MaxTotalBytes: helper.svcCtx.Config.Waf.ExtractMaxTotalBytes,
	}); err != nil {
		_ = os.RemoveAll(releaseDir)
		helper.finishJob(job, wafJobStatusFailed, err.Error(), 0)
		return nil, err
	}

	release := &model.WafRelease{
		SourceID:     0,
		Kind:         kind,
		Version:      version,
		ArtifactType: artifactTypeFromExt(verifyResult.Ext),
		Checksum:     verifyResult.SHA256,
		SizeBytes:    verifyResult.SizeBytes,
		StoragePath:  filepath.Clean(releaseDir),
		Status:       wafReleaseStatusVerified,
		Meta:         model.JSONMap{"originFileName": basenameSafe(fileName)},
	}

	if err := helper.svcCtx.DB.Create(release).Error; err != nil {
		helper.finishJob(job, wafJobStatusFailed, fmt.Sprintf("create release failed: %v", err), 0)
		return nil, fmt.Errorf("create release failed: %w", err)
	}
	helper.applyReleaseRetention(release.Kind)

	helper.finishJob(job, wafJobStatusSuccess, "upload success", release.ID)

	if kind != wafKindCorazaEngine && req.ActivateNow {
		activateLogic := NewActivateWafReleaseLogic(l.ctx, l.svcCtx)
		if _, activateErr := activateLogic.ActivateWafRelease(&types.WafReleaseActivateReq{ID: release.ID}); activateErr != nil {
			return nil, activateErr
		}
	}

	return &types.BaseResp{Code: 200, Msg: "success"}, nil
}
