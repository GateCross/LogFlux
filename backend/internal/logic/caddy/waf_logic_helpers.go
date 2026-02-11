package caddy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
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
	wafPolicyBoolMaskCtxKey = "waf_policy_bool_mask"
)

type wafLogicHelper struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logger logx.Logger
	store  *waf.Store
}

func newWafLogicHelper(ctx context.Context, svcCtx *svc.ServiceContext, logger logx.Logger) *wafLogicHelper {
	workDir := strings.TrimSpace(svcCtx.Config.Waf.WorkDir)
	if workDir == "" {
		workDir = "/config/security"
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

func normalizeWafKind(kind string) string {
	normalized := strings.ToLower(strings.TrimSpace(kind))
	if normalized == "" {
		return wafKindCRS
	}
	return normalized
}

func validateWafKind(kind string) error {
	switch normalizeWafKind(kind) {
	case wafKindCRS, wafKindCorazaEngine:
		return nil
	default:
		return fmt.Errorf("invalid kind: %s", kind)
	}
}

func normalizeWafMode(mode string) string {
	normalized := strings.ToLower(strings.TrimSpace(mode))
	if normalized == "" {
		return wafModeRemote
	}
	return normalized
}

func validateWafMode(mode string) error {
	switch normalizeWafMode(mode) {
	case wafModeRemote, wafModeManual:
		return nil
	default:
		return fmt.Errorf("invalid mode: %s", mode)
	}
}

func normalizeWafAuthType(authType string) string {
	normalized := strings.ToLower(strings.TrimSpace(authType))
	if normalized == "" {
		return wafAuthNone
	}
	return normalized
}

func validateWafAuthType(authType string) error {
	switch normalizeWafAuthType(authType) {
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

func (helper *wafLogicHelper) startJob(sourceID, releaseID uint, action, triggerMode string) *model.WafUpdateJob {
	now := time.Now()
	job := &model.WafUpdateJob{
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

func (helper *wafLogicHelper) finishJob(job *model.WafUpdateJob, status, message string, releaseID uint) {
	if job == nil {
		return
	}

	now := time.Now()
	localizedMessage := localizeWafJobMessage(strings.TrimSpace(message))
	updates := map[string]interface{}{
		"status":      strings.ToLower(strings.TrimSpace(status)),
		"message":     localizedMessage,
		"finished_at": &now,
	}
	if releaseID > 0 {
		updates["release_id"] = releaseID
	}
	if err := helper.svcCtx.DB.Model(job).Updates(updates).Error; err != nil {
		helper.logger.Errorf("finish waf job failed: %v", err)
	}
}

func localizeWafJobMessage(rawMessage string) string {
	messageText := strings.TrimSpace(rawMessage)
	if messageText == "" {
		return ""
	}

	exactMap := map[string]string{
		"check success":               "检查成功",
		"sync success":                "同步成功",
		"upload success":              "上传成功",
		"activate success":            "激活成功",
		"rollback success":            "回滚成功",
		"engine source check success": "引擎源检查成功",
	}
	if localized, ok := exactMap[messageText]; ok {
		return localized
	}

	replacementRules := []struct {
		pattern     *regexp.Regexp
		replacement string
	}{
		{regexp.MustCompile(`(?i)context deadline exceeded`), "请求超时"},
		{regexp.MustCompile(`(?i)i/o timeout`), "网络超时"},
		{regexp.MustCompile(`(?i)invalid proxy url:`), "代理地址不合法："},
		{regexp.MustCompile(`(?i)invalid url:`), "无效地址："},
		{regexp.MustCompile(`(?i)only https url is allowed`), "仅支持 HTTPS 地址"},
		{regexp.MustCompile(`(?i)only https scheme is allowed`), "仅允许 HTTPS 协议"},
		{regexp.MustCompile(`(?i)proxy url scheme must be http or https`), "代理地址协议仅支持 http/https"},
		{regexp.MustCompile(`(?i)source not found`), "未找到更新源"},
		{regexp.MustCompile(`(?i)source is disabled`), "更新源已禁用"},
		{regexp.MustCompile(`(?i)source mode is not remote`), "更新源模式不是 remote"},
		{regexp.MustCompile(`(?i)source url is empty`), "更新源地址为空"},
		{regexp.MustCompile(`(?i)move package failed:`), "移动安装包失败："},
		{regexp.MustCompile(`(?i)create release dir failed:`), "创建版本目录失败："},
		{regexp.MustCompile(`(?i)create release failed:`), "创建版本记录失败："},
		{regexp.MustCompile(`(?i)fetch failed:`), "下载失败："},
		{regexp.MustCompile(`(?i)host not allowed:`), "源域名不在允许列表："},
		{regexp.MustCompile(`(?i)unexpected status code:`), "下载返回异常状态码："},
		{regexp.MustCompile(`(?i)write temp file failed:`), "写入临时文件失败："},
		{regexp.MustCompile(`(?i)close temp file failed:`), "关闭临时文件失败："},
		{regexp.MustCompile(`(?i)move temp file failed:`), "移动临时文件失败："},
		{regexp.MustCompile(`(?i)prepare waf store failed:`), "准备 Waf 存储目录失败："},
	}

	localized := messageText
	for _, rule := range replacementRules {
		localized = rule.pattern.ReplaceAllString(localized, rule.replacement)
	}

	return strings.TrimSpace(localized)
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

func (helper *wafLogicHelper) policyBoolMask() map[string]bool {
	rawMask := helper.ctx.Value(wafPolicyBoolMaskCtxKey)
	if rawMask == nil {
		return nil
	}
	mask, ok := rawMask.(map[string]bool)
	if !ok {
		return nil
	}
	return mask
}

func (helper *wafLogicHelper) hasPolicyBoolField(field string) bool {
	mask := helper.policyBoolMask()
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

func (helper *wafLogicHelper) activateRelease(release *model.WafRelease) error {
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

func (helper *wafLogicHelper) markReleaseActive(release *model.WafRelease) error {
	if release == nil {
		return fmt.Errorf("release is nil")
	}

	return helper.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.WafRelease{}).
			Where("kind = ? AND status = ? AND id <> ?", release.Kind, wafReleaseStatusActive, release.ID).
			Update("status", wafReleaseStatusRolledBack).Error; err != nil {
			return err
		}

		if err := tx.Model(&model.WafRelease{}).
			Where("id = ?", release.ID).
			Updates(map[string]interface{}{
				"status": wafReleaseStatusActive,
			}).Error; err != nil {
			return err
		}

		if release.SourceID > 0 {
			if err := tx.Model(&model.WafSource{}).
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

func (helper *wafLogicHelper) markReleaseFailed(release *model.WafRelease, message string) {
	if release == nil {
		return
	}

	errorMessage := strings.TrimSpace(message)
	if errorMessage == "" {
		errorMessage = "activate failed"
	}

	if err := helper.svcCtx.DB.Model(&model.WafRelease{}).
		Where("id = ?", release.ID).
		Updates(map[string]interface{}{"status": wafReleaseStatusFailed}).Error; err != nil {
		helper.logger.Errorf("mark release failed status error: %v", err)
	}

	if release.SourceID > 0 {
		if err := helper.svcCtx.DB.Model(&model.WafSource{}).
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
	if err := helper.svcCtx.DB.Model(&model.WafSource{}).
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
	if err := helper.svcCtx.DB.Model(&model.WafSource{}).Where("id = ?", sourceID).Updates(updates).Error; err != nil {
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

func findLatestReleaseByKindAndVersion(db *gorm.DB, kind, version string) (*model.WafRelease, error) {
	candidateKind := normalizeWafKind(kind)
	candidateVersion := strings.TrimSpace(version)
	if candidateVersion == "" {
		return nil, fmt.Errorf("version is required")
	}

	var release model.WafRelease
	err := db.Where("kind = ? AND version = ?", candidateKind, candidateVersion).
		Order("id desc").
		First(&release).Error
	if err == nil {
		return &release, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return nil, fmt.Errorf("query release version failed: %w", err)
}

func (helper *wafLogicHelper) ensurePathInWorkDir(pathValue string) (string, error) {
	baseDir := filepath.Clean(strings.TrimSpace(helper.store.BaseDir))
	cleanPath := filepath.Clean(strings.TrimSpace(pathValue))
	if cleanPath == "" {
		return "", fmt.Errorf("path is empty")
	}

	if cleanPath == baseDir {
		return cleanPath, nil
	}

	prefix := baseDir + string(os.PathSeparator)
	if !strings.HasPrefix(cleanPath, prefix) {
		return "", fmt.Errorf("仅允许读取 %s 目录内的文件", baseDir)
	}

	return cleanPath, nil
}

func (helper *wafLogicHelper) canReuseRelease(release *model.WafRelease) bool {
	if release == nil {
		return false
	}

	releaseDir := helper.store.ReleaseDir(release.Version)
	if _, err := helper.ensurePathInWorkDir(releaseDir); err != nil {
		return false
	}

	stat, err := os.Stat(releaseDir)
	if err != nil {
		return false
	}
	return stat.IsDir()
}

func basenameSafe(pathValue string) string {
	base := filepath.Base(strings.TrimSpace(pathValue))
	if base == "." || base == "/" || base == "" {
		return "package"
	}
	return base
}
