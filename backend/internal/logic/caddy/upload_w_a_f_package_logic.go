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

type UploadWAFPackageLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUploadWAFPackageLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UploadWAFPackageLogic {
	return &UploadWAFPackageLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UploadWAFPackageLogic) UploadWAFPackage(req *types.WAFUploadReq) (resp *types.BaseResp, err error) {
	helper := newWAFLogicHelper(l.ctx, l.svcCtx, l.Logger)

	if err := helper.ensureStoreDirs(); err != nil {
		return nil, err
	}

	kind := normalizeWAFKind(req.Kind)
	if err := validateWAFKind(kind); err != nil {
		return nil, err
	}

	version := strings.TrimSpace(req.Version)
	if version == "" {
		return nil, fmt.Errorf("version is required")
	}
	version = sanitizeToken(version)
	version = ensureUniqueReleaseVersion(helper.svcCtx.DB, 0, version)

	tempPath, _ := l.ctx.Value(wafUploadTempPathCtxKey).(string)
	tempPath = strings.TrimSpace(tempPath)
	if tempPath == "" {
		return nil, fmt.Errorf("upload file is required")
	}

	fileName, _ := l.ctx.Value(wafUploadFileNameCtxKey).(string)
	if strings.TrimSpace(fileName) == "" {
		fileName = basenameSafe(tempPath)
	}

	job := helper.startJob(0, 0, "verify", "upload")
	defer func() {
		if strings.TrimSpace(tempPath) != "" {
			_ = os.Remove(tempPath)
		}
	}()

	verifyResult, err := waf.VerifyPackage(tempPath, waf.VerifyOptions{
		AllowedExt:      []string{".tar.gz", ".zip"},
		MaxPackageBytes: helper.svcCtx.Config.WAF.MaxPackageBytes,
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
		MaxFiles:      helper.svcCtx.Config.WAF.ExtractMaxFiles,
		MaxTotalBytes: helper.svcCtx.Config.WAF.ExtractMaxTotalBytes,
	}); err != nil {
		_ = os.RemoveAll(releaseDir)
		helper.finishJob(job, wafJobStatusFailed, err.Error(), 0)
		return nil, err
	}

	release := &model.WAFRelease{
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

	helper.finishJob(job, wafJobStatusSuccess, "upload success", release.ID)

	if req.ActivateNow {
		activateLogic := NewActivateWAFReleaseLogic(l.ctx, l.svcCtx)
		if _, activateErr := activateLogic.ActivateWAFRelease(&types.WAFReleaseActivateReq{ID: release.ID}); activateErr != nil {
			return nil, activateErr
		}
	}

	return &types.BaseResp{Code: 200, Msg: "success"}, nil
}
