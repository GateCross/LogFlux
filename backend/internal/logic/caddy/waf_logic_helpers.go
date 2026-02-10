package caddy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"logflux/internal/svc"
	"logflux/internal/waf"
	"logflux/model"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

const (
	wafKindCRS          = "crs"
	wafKindCorazaEngine = "coraza_engine"

	wafModeRemote = "remote"
	wafModeManual = "manual"

	wafAuthNone  = "none"
	wafAuthToken = "token"
	wafAuthBasic = "basic"

	wafReleaseStatusDownloaded = "downloaded"
	wafReleaseStatusVerified   = "verified"
	wafReleaseStatusActive     = "active"
	wafReleaseStatusFailed     = "failed"
	wafReleaseStatusRolledBack = "rolled_back"

	wafJobStatusRunning = "running"
	wafJobStatusSuccess = "success"
	wafJobStatusFailed  = "failed"

	wafUploadTempPathCtxKey = "waf_upload_temp_path"
	wafUploadFileNameCtxKey = "waf_upload_file_name"
	wafSourceBoolMaskCtxKey = "waf_source_bool_mask"
)

type wafLogicHelper struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logger logx.Logger
	store  *waf.Store
}

func newWAFLogicHelper(ctx context.Context, svcCtx *svc.ServiceContext, logger logx.Logger) *wafLogicHelper {
	workDir := strings.TrimSpace(svcCtx.Config.WAF.WorkDir)
	if workDir == "" {
		workDir = "/config/caddy/waf"
	}
	return &wafLogicHelper{
		ctx:    ctx,
		svcCtx: svcCtx,
		logger: logger,
		store:  waf.NewStore(workDir),
	}
}

func (helper *wafLogicHelper) ensureStoreDirs() error {
	if err := helper.store.EnsureDirs(); err != nil {
		return fmt.Errorf("prepare waf store failed: %w", err)
	}
	return nil
}

func normalizeWAFKind(kind string) string {
	normalized := strings.ToLower(strings.TrimSpace(kind))
	if normalized == "" {
		return wafKindCRS
	}
	return normalized
}

func validateWAFKind(kind string) error {
	switch normalizeWAFKind(kind) {
	case wafKindCRS, wafKindCorazaEngine:
		return nil
	default:
		return fmt.Errorf("invalid kind: %s", kind)
	}
}

func normalizeWAFMode(mode string) string {
	normalized := strings.ToLower(strings.TrimSpace(mode))
	if normalized == "" {
		return wafModeRemote
	}
	return normalized
}

func validateWAFMode(mode string) error {
	switch normalizeWAFMode(mode) {
	case wafModeRemote, wafModeManual:
		return nil
	default:
		return fmt.Errorf("invalid mode: %s", mode)
	}
}

func normalizeWAFAuthType(authType string) string {
	normalized := strings.ToLower(strings.TrimSpace(authType))
	if normalized == "" {
		return wafAuthNone
	}
	return normalized
}

func validateWAFAuthType(authType string) error {
	switch normalizeWAFAuthType(authType) {
	case wafAuthNone, wafAuthToken, wafAuthBasic:
		return nil
	default:
		return fmt.Errorf("invalid auth type: %s", authType)
	}
}

func parseMetaJSON(raw string) (model.JSONMap, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}

	decoded := make(map[string]interface{})
	if err := json.Unmarshal([]byte(trimmed), &decoded); err != nil {
		return nil, fmt.Errorf("invalid meta json: %w", err)
	}
	return model.JSONMap(decoded), nil
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format("2006-01-02 15:04:05")
}

func formatNullableTime(value *time.Time) string {
	if value == nil {
		return ""
	}
	return formatTime(*value)
}

func (helper *wafLogicHelper) startJob(sourceID, releaseID uint, action, triggerMode string) *model.WAFUpdateJob {
	now := time.Now()
	job := &model.WAFUpdateJob{
		SourceID:    sourceID,
		ReleaseID:   releaseID,
		Action:      strings.ToLower(strings.TrimSpace(action)),
		TriggerMode: strings.ToLower(strings.TrimSpace(triggerMode)),
		Operator:    helper.currentOperator(),
		Status:      wafJobStatusRunning,
		StartedAt:   &now,
		FinishedAt:  nil,
		Message:     "",
	}
	if job.TriggerMode == "" {
		job.TriggerMode = "manual"
	}
	if err := helper.svcCtx.DB.Create(job).Error; err != nil {
		helper.logger.Errorf("create waf job failed: %v", err)
		return nil
	}
	return job
}

func (helper *wafLogicHelper) finishJob(job *model.WAFUpdateJob, status, message string, releaseID uint) {
	if job == nil {
		return
	}

	now := time.Now()
	updates := map[string]interface{}{
		"status":      strings.ToLower(strings.TrimSpace(status)),
		"message":     strings.TrimSpace(message),
		"finished_at": &now,
	}
	if releaseID > 0 {
		updates["release_id"] = releaseID
	}
	if err := helper.svcCtx.DB.Model(job).Updates(updates).Error; err != nil {
		helper.logger.Errorf("finish waf job failed: %v", err)
	}
}

func (helper *wafLogicHelper) sourceBoolMask() map[string]bool {
	rawMask := helper.ctx.Value(wafSourceBoolMaskCtxKey)
	if rawMask == nil {
		return nil
	}
	mask, ok := rawMask.(map[string]bool)
	if !ok {
		return nil
	}
	return mask
}

func (helper *wafLogicHelper) hasSourceBoolField(field string) bool {
	mask := helper.sourceBoolMask()
	if len(mask) == 0 {
		return false
	}
	return mask[field]
}

func (helper *wafLogicHelper) currentOperator() string {
	userID := helper.ctx.Value("userId")
	switch value := userID.(type) {
	case nil:
		return "system"
	case string:
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return "system"
		}
		return trimmed
	case int:
		return strconv.Itoa(value)
	case int32:
		return strconv.FormatInt(int64(value), 10)
	case int64:
		return strconv.FormatInt(value, 10)
	case uint:
		return strconv.FormatUint(uint64(value), 10)
	case uint32:
		return strconv.FormatUint(uint64(value), 10)
	case uint64:
		return strconv.FormatUint(value, 10)
	case float32:
		return strconv.FormatInt(int64(value), 10)
	case float64:
		return strconv.FormatInt(int64(value), 10)
	default:
		return fmt.Sprintf("%v", value)
	}
}

func (helper *wafLogicHelper) primaryCaddyServer() (*model.CaddyServer, error) {
	var server model.CaddyServer
	err := helper.svcCtx.DB.Where("type = ?", "local").Order("id asc").First(&server).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		err = helper.svcCtx.DB.Order("id asc").First(&server).Error
	}
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("caddy server not found")
		}
		return nil, fmt.Errorf("query caddy server failed: %w", err)
	}
	if strings.TrimSpace(server.Config) == "" {
		return nil, fmt.Errorf("caddy config is empty, please save caddy config first")
	}
	return &server, nil
}

type wafCaddyLoader struct {
	server *model.CaddyServer
}

func (loader *wafCaddyLoader) Adapt(config string) error {
	return adaptCaddyfile(loader.server, config)
}

func (loader *wafCaddyLoader) Load(config string) error {
	return loadCaddyfile(loader.server, config)
}

func (helper *wafLogicHelper) activateRelease(release *model.WAFRelease) error {
	if release == nil {
		return fmt.Errorf("release is nil")
	}
	if err := helper.ensureStoreDirs(); err != nil {
		return err
	}

	server, err := helper.primaryCaddyServer()
	if err != nil {
		return err
	}

	activator := &waf.Activator{
		Store:       helper.store,
		CaddyLoader: &wafCaddyLoader{server: server},
	}
	if err := activator.ActivateVersion(release.Version, server.Config); err != nil {
		return fmt.Errorf("activate version failed: %w", err)
	}
	return nil
}

func (helper *wafLogicHelper) markReleaseActive(release *model.WAFRelease) error {
	if release == nil {
		return fmt.Errorf("release is nil")
	}

	return helper.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.WAFRelease{}).
			Where("kind = ? AND status = ? AND id <> ?", release.Kind, wafReleaseStatusActive, release.ID).
			Update("status", wafReleaseStatusRolledBack).Error; err != nil {
			return err
		}

		if err := tx.Model(&model.WAFRelease{}).
			Where("id = ?", release.ID).
			Updates(map[string]interface{}{
				"status": wafReleaseStatusActive,
			}).Error; err != nil {
			return err
		}

		if release.SourceID > 0 {
			if err := tx.Model(&model.WAFSource{}).
				Where("id = ?", release.SourceID).
				Updates(map[string]interface{}{
					"last_release": release.Version,
					"last_error":   "",
				}).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

func (helper *wafLogicHelper) markReleaseFailed(release *model.WAFRelease, message string) {
	if release == nil {
		return
	}

	errorMessage := strings.TrimSpace(message)
	if errorMessage == "" {
		errorMessage = "activate failed"
	}

	if err := helper.svcCtx.DB.Model(&model.WAFRelease{}).
		Where("id = ?", release.ID).
		Updates(map[string]interface{}{"status": wafReleaseStatusFailed}).Error; err != nil {
		helper.logger.Errorf("mark release failed status error: %v", err)
	}

	if release.SourceID > 0 {
		if err := helper.svcCtx.DB.Model(&model.WAFSource{}).
			Where("id = ?", release.SourceID).
			Updates(map[string]interface{}{"last_error": errorMessage}).Error; err != nil {
			helper.logger.Errorf("update source last_error failed: %v", err)
		}
	}
}

func (helper *wafLogicHelper) clearSourceError(sourceID uint) {
	if sourceID == 0 {
		return
	}
	if err := helper.svcCtx.DB.Model(&model.WAFSource{}).
		Where("id = ?", sourceID).
		Update("last_error", "").Error; err != nil {
		helper.logger.Errorf("clear source last_error failed: %v", err)
	}
}

func (helper *wafLogicHelper) updateSourceLastCheck(sourceID uint, releaseVersion, errMessage string) {
	if sourceID == 0 {
		return
	}
	now := time.Now()
	updates := map[string]interface{}{
		"last_checked_at": &now,
		"last_error":      strings.TrimSpace(errMessage),
	}
	if strings.TrimSpace(releaseVersion) != "" {
		updates["last_release"] = strings.TrimSpace(releaseVersion)
	}
	if err := helper.svcCtx.DB.Model(&model.WAFSource{}).Where("id = ?", sourceID).Updates(updates).Error; err != nil {
		helper.logger.Errorf("update source last check failed: %v", err)
	}
}

func detectPackageExt(fileName string) string {
	lower := strings.ToLower(strings.TrimSpace(fileName))
	switch {
	case strings.HasSuffix(lower, ".tar.gz"):
		return ".tar.gz"
	case strings.HasSuffix(lower, ".zip"):
		return ".zip"
	default:
		return ""
	}
}

func artifactTypeFromExt(ext string) string {
	switch strings.ToLower(strings.TrimSpace(ext)) {
	case ".zip":
		return "zip"
	case ".tar.gz":
		return "tar.gz"
	default:
		return "upload"
	}
}

func sanitizeToken(raw string) string {
	replacer := strings.NewReplacer("/", "_", "\\", "_", " ", "_", "..", "_")
	cleaned := replacer.Replace(strings.TrimSpace(raw))
	if cleaned == "" {
		return "unknown"
	}
	return cleaned
}

func deriveVersionFromURL(downloadURL string) string {
	parsed, err := url.Parse(strings.TrimSpace(downloadURL))
	if err != nil {
		return time.Now().Format("20060102_150405")
	}
	base := path.Base(parsed.Path)
	if base == "." || base == "/" || base == "" {
		return time.Now().Format("20060102_150405")
	}
	ext := detectPackageExt(base)
	version := base
	if ext != "" {
		version = strings.TrimSuffix(base, ext)
	}
	version = sanitizeToken(version)
	if version == "unknown" {
		return time.Now().Format("20060102_150405")
	}
	return version
}

func ensureUniqueReleaseVersion(db *gorm.DB, sourceID uint, version string) string {
	candidate := strings.TrimSpace(version)
	if candidate == "" {
		candidate = time.Now().Format("20060102_150405")
	}

	var count int64
	if err := db.Model(&model.WAFRelease{}).
		Where("source_id = ? AND version = ?", sourceID, candidate).
		Count(&count).Error; err != nil {
		return candidate
	}
	if count == 0 {
		return candidate
	}
	return fmt.Sprintf("%s_%d", candidate, time.Now().Unix())
}

func basenameSafe(pathValue string) string {
	base := filepath.Base(strings.TrimSpace(pathValue))
	if base == "." || base == "/" || base == "" {
		return "package"
	}
	return base
}
