package caddy

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"logflux/internal/types"
	"logflux/model"

	"gorm.io/gorm"
)

const (
	wafPolicyStatusDraft      = "draft"
	wafPolicyStatusPublished  = "published"
	wafPolicyStatusRolledBack = "rolled_back"
)

func parsePolicyConfigJSON(raw string) (model.JSONMap, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}

	decoded := make(map[string]interface{})
	if err := json.Unmarshal([]byte(trimmed), &decoded); err != nil {
		return nil, fmt.Errorf("策略配置 JSON 无效: %w", err)
	}
	return model.JSONMap(decoded), nil
}

func marshalPolicyConfigJSON(config model.JSONMap) string {
	if len(config) == 0 {
		return ""
	}
	bytes, err := json.Marshal(config)
	if err != nil {
		return ""
	}
	return string(bytes)
}

func normalizePolicyName(name string) string {
	return strings.TrimSpace(name)
}

func applyPolicyReqToModel(helper *wafLogicHelper, req *types.WafPolicyReq, policy *model.WafPolicy) error {
	if req == nil || policy == nil {
		return fmt.Errorf("策略参数不合法")
	}

	name := normalizePolicyName(req.Name)
	if name == "" {
		return fmt.Errorf("策略名称不能为空")
	}

	if err := validatePolicyEngineMode(req.EngineMode); err != nil {
		return err
	}
	if err := validatePolicyAuditEngine(req.AuditEngine); err != nil {
		return err
	}
	if err := validatePolicyAuditLogFormat(req.AuditLogFormat); err != nil {
		return err
	}

	requestBodyLimit := normalizePolicyRequestBodyLimit(req.RequestBodyLimit, 10*1024*1024)
	requestBodyNoFilesLimit := normalizePolicyRequestBodyLimit(req.RequestBodyNoFilesLimit, 1024*1024)
	if err := validatePolicyRequestBodyLimit(requestBodyLimit, "requestBodyLimit"); err != nil {
		return err
	}
	if err := validatePolicyRequestBodyLimit(requestBodyNoFilesLimit, "requestBodyNoFilesLimit"); err != nil {
		return err
	}

	config, err := parsePolicyConfigJSON(req.Config)
	if err != nil {
		return err
	}

	crsTemplate := normalizePolicyCRSTemplate(req.CrsTemplate)
	if err := validatePolicyCRSTemplate(crsTemplate); err != nil {
		return err
	}
	preset := defaultPolicyCRSTuningByTemplate(crsTemplate)
	crsParanoiaLevel := req.CrsParanoiaLevel
	if crsParanoiaLevel <= 0 {
		crsParanoiaLevel = preset.ParanoiaLevel
	}
	if err := validatePolicyCRSParanoiaLevel(crsParanoiaLevel); err != nil {
		return err
	}
	crsInboundAnomalyThreshold := req.CrsInboundAnomalyThreshold
	if crsInboundAnomalyThreshold <= 0 {
		crsInboundAnomalyThreshold = preset.InboundAnomalyThreshold
	}
	if err := validatePolicyCRSAnomalyThreshold(crsInboundAnomalyThreshold, "crsInboundAnomalyThreshold"); err != nil {
		return err
	}
	crsOutboundAnomalyThreshold := req.CrsOutboundAnomalyThreshold
	if crsOutboundAnomalyThreshold <= 0 {
		crsOutboundAnomalyThreshold = preset.OutboundAnomalyThreshold
	}
	if err := validatePolicyCRSAnomalyThreshold(crsOutboundAnomalyThreshold, "crsOutboundAnomalyThreshold"); err != nil {
		return err
	}

	policy.Name = name
	policy.Description = strings.TrimSpace(req.Description)
	policy.EngineMode = normalizePolicyEngineMode(req.EngineMode)
	policy.AuditEngine = normalizePolicyAuditEngine(req.AuditEngine)
	policy.AuditLogFormat = normalizePolicyAuditLogFormat(req.AuditLogFormat)
	policy.AuditRelevantStatus = normalizePolicyAuditRelevantStatus(req.AuditRelevantStatus)
	policy.RequestBodyLimit = requestBodyLimit
	policy.RequestBodyNoFilesLimit = requestBodyNoFilesLimit
	policy.CrsTemplate = crsTemplate
	policy.CrsParanoiaLevel = crsParanoiaLevel
	policy.CrsInboundAnomalyThreshold = crsInboundAnomalyThreshold
	policy.CrsOutboundAnomalyThreshold = crsOutboundAnomalyThreshold
	policy.Config = config

	if policy.ID == 0 {
		policy.Enabled = true
		policy.IsDefault = false
		policy.RequestBodyAccess = true
	}

	if helper.hasPolicyBoolField("enabled") {
		policy.Enabled = req.Enabled
	}
	if helper.hasPolicyBoolField("isDefault") {
		policy.IsDefault = req.IsDefault
	}
	if helper.hasPolicyBoolField("requestBodyAccess") {
		policy.RequestBodyAccess = req.RequestBodyAccess
	}

	if err := ensurePolicyCRSTuning(policy); err != nil {
		return err
	}

	return nil
}

func applyPolicyUpdateReqToModel(helper *wafLogicHelper, req *types.WafPolicyUpdateReq, policy *model.WafPolicy) error {
	if req == nil || policy == nil {
		return fmt.Errorf("策略参数不合法")
	}

	if name := normalizePolicyName(req.Name); name != "" {
		policy.Name = name
	}
	if desc := strings.TrimSpace(req.Description); desc != "" {
		policy.Description = desc
	}

	if strings.TrimSpace(req.EngineMode) != "" {
		if err := validatePolicyEngineMode(req.EngineMode); err != nil {
			return err
		}
		policy.EngineMode = normalizePolicyEngineMode(req.EngineMode)
	}
	if strings.TrimSpace(req.AuditEngine) != "" {
		if err := validatePolicyAuditEngine(req.AuditEngine); err != nil {
			return err
		}
		policy.AuditEngine = normalizePolicyAuditEngine(req.AuditEngine)
	}
	if strings.TrimSpace(req.AuditLogFormat) != "" {
		if err := validatePolicyAuditLogFormat(req.AuditLogFormat); err != nil {
			return err
		}
		policy.AuditLogFormat = normalizePolicyAuditLogFormat(req.AuditLogFormat)
	}
	if strings.TrimSpace(req.AuditRelevantStatus) != "" {
		policy.AuditRelevantStatus = normalizePolicyAuditRelevantStatus(req.AuditRelevantStatus)
	}

	if req.RequestBodyLimit > 0 {
		if err := validatePolicyRequestBodyLimit(req.RequestBodyLimit, "requestBodyLimit"); err != nil {
			return err
		}
		policy.RequestBodyLimit = req.RequestBodyLimit
	}
	if req.RequestBodyNoFilesLimit > 0 {
		if err := validatePolicyRequestBodyLimit(req.RequestBodyNoFilesLimit, "requestBodyNoFilesLimit"); err != nil {
			return err
		}
		policy.RequestBodyNoFilesLimit = req.RequestBodyNoFilesLimit
	}
	if strings.TrimSpace(req.CrsTemplate) != "" {
		if err := validatePolicyCRSTemplate(req.CrsTemplate); err != nil {
			return err
		}
		policy.CrsTemplate = normalizePolicyCRSTemplate(req.CrsTemplate)
	}
	if req.CrsParanoiaLevel > 0 {
		if err := validatePolicyCRSParanoiaLevel(req.CrsParanoiaLevel); err != nil {
			return err
		}
		policy.CrsParanoiaLevel = req.CrsParanoiaLevel
	}
	if req.CrsInboundAnomalyThreshold > 0 {
		if err := validatePolicyCRSAnomalyThreshold(req.CrsInboundAnomalyThreshold, "crsInboundAnomalyThreshold"); err != nil {
			return err
		}
		policy.CrsInboundAnomalyThreshold = req.CrsInboundAnomalyThreshold
	}
	if req.CrsOutboundAnomalyThreshold > 0 {
		if err := validatePolicyCRSAnomalyThreshold(req.CrsOutboundAnomalyThreshold, "crsOutboundAnomalyThreshold"); err != nil {
			return err
		}
		policy.CrsOutboundAnomalyThreshold = req.CrsOutboundAnomalyThreshold
	}

	if strings.TrimSpace(req.Config) != "" {
		config, err := parsePolicyConfigJSON(req.Config)
		if err != nil {
			return err
		}
		policy.Config = config
	}

	if helper.hasPolicyBoolField("enabled") {
		policy.Enabled = req.Enabled
	}
	if helper.hasPolicyBoolField("isDefault") {
		policy.IsDefault = req.IsDefault
	}
	if helper.hasPolicyBoolField("requestBodyAccess") {
		policy.RequestBodyAccess = req.RequestBodyAccess
	}

	if strings.TrimSpace(policy.Name) == "" {
		return fmt.Errorf("策略名称不能为空")
	}
	if strings.TrimSpace(policy.EngineMode) == "" {
		policy.EngineMode = "on"
	}
	if strings.TrimSpace(policy.AuditEngine) == "" {
		policy.AuditEngine = "relevantonly"
	}
	if strings.TrimSpace(policy.AuditLogFormat) == "" {
		policy.AuditLogFormat = "json"
	}
	policy.AuditRelevantStatus = normalizePolicyAuditRelevantStatus(policy.AuditRelevantStatus)

	if err := validatePolicyEngineMode(policy.EngineMode); err != nil {
		return err
	}
	if err := validatePolicyAuditEngine(policy.AuditEngine); err != nil {
		return err
	}
	if err := validatePolicyAuditLogFormat(policy.AuditLogFormat); err != nil {
		return err
	}
	if err := validatePolicyRequestBodyLimit(policy.RequestBodyLimit, "requestBodyLimit"); err != nil {
		return err
	}
	if err := validatePolicyRequestBodyLimit(policy.RequestBodyNoFilesLimit, "requestBodyNoFilesLimit"); err != nil {
		return err
	}
	if err := ensurePolicyCRSTuning(policy); err != nil {
		return err
	}

	return nil
}

func ensureSingleDefaultPolicy(tx *gorm.DB, policy *model.WafPolicy) error {
	if tx == nil || policy == nil {
		return fmt.Errorf("策略事务无效")
	}
	if !policy.IsDefault {
		return nil
	}

	query := tx.Model(&model.WafPolicy{}).Where("is_default = ?", true)
	if policy.ID > 0 {
		query = query.Where("id <> ?", policy.ID)
	}
	if err := query.Update("is_default", false).Error; err != nil {
		return fmt.Errorf("重置默认策略失败: %w", err)
	}
	return nil
}

func createPolicyRevision(tx *gorm.DB, policy *model.WafPolicy, status, directives, message, operator string) (*model.WafPolicyRevision, error) {
	if tx == nil || policy == nil {
		return nil, fmt.Errorf("策略版本上下文无效")
	}

	var lastRevision model.WafPolicyRevision
	nextVersion := uint(1)
	if err := tx.Where("policy_id = ?", policy.ID).Order("version desc").First(&lastRevision).Error; err == nil {
		nextVersion = lastRevision.Version + 1
	} else if err != nil && err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("查询最新策略版本失败: %w", err)
	}

	revision := &model.WafPolicyRevision{
		PolicyID:           policy.ID,
		Version:            nextVersion,
		Status:             strings.TrimSpace(status),
		ConfigSnapshot:     policy.Config,
		DirectivesSnapshot: strings.TrimSpace(directives),
		Operator:           strings.TrimSpace(operator),
		Message:            strings.TrimSpace(message),
	}
	if revision.Status == "" {
		revision.Status = wafPolicyStatusDraft
	}

	if err := tx.Create(revision).Error; err != nil {
		return nil, fmt.Errorf("创建策略版本失败: %w", err)
	}

	return revision, nil
}

func markPolicyRevisionsRolledBack(tx *gorm.DB, policyID uint, excludeRevisionID uint) error {
	if tx == nil || policyID == 0 {
		return nil
	}
	query := tx.Model(&model.WafPolicyRevision{}).
		Where("policy_id = ? AND status = ?", policyID, wafPolicyStatusPublished)
	if excludeRevisionID > 0 {
		query = query.Where("id <> ?", excludeRevisionID)
	}
	if err := query.Update("status", wafPolicyStatusRolledBack).Error; err != nil {
		return fmt.Errorf("标记策略版本已回滚失败: %w", err)
	}
	return nil
}

func findPrimaryCaddyServer(db *gorm.DB) (*model.CaddyServer, error) {
	if db == nil {
		return nil, fmt.Errorf("数据库为空")
	}

	var server model.CaddyServer
	err := db.Where("type = ?", "local").Order("id asc").First(&server).Error
	if err == gorm.ErrRecordNotFound {
		err = db.Order("id asc").First(&server).Error
	}
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("Caddy 服务器不存在")
		}
		return nil, fmt.Errorf("查询 Caddy 服务器失败: %w", err)
	}
	if strings.TrimSpace(server.Config) == "" {
		return nil, fmt.Errorf("Caddy 配置为空，请先保存 Caddy 配置")
	}
	return &server, nil
}

func currentOperatorFromContext(ctx context.Context) string {
	if ctx == nil {
		return "system"
	}
	userID := ctx.Value("userId")
	if userID == nil {
		return "system"
	}
	value := strings.TrimSpace(fmt.Sprintf("%v", userID))
	if value == "" {
		return "system"
	}
	return value
}

func normalizeCaddyModulesJSON(modules string) string {
	trimmed := strings.TrimSpace(modules)
	if trimmed == "" {
		return emptyModulesJSON
	}
	return trimmed
}

func createCaddyPolicyHistory(tx *gorm.DB, serverID uint, action, config, modules string) error {
	if tx == nil {
		return fmt.Errorf("数据库为空")
	}

	history := &model.CaddyConfigHistory{
		ServerID: serverID,
		Action:   strings.TrimSpace(action),
		Hash:     hashConfig(config),
		Config:   config,
		Modules:  normalizeCaddyModulesJSON(modules),
	}
	if err := tx.Create(history).Error; err != nil {
		return fmt.Errorf("创建 Caddy 配置历史失败: %w", err)
	}
	return nil
}

func rollbackPolicyConfigToLastGood(server *model.CaddyServer, lastGoodConfig string) error {
	if server == nil {
		return fmt.Errorf("Caddy 服务器不存在")
	}
	rollbackConfig := strings.TrimSpace(lastGoodConfig)
	if rollbackConfig == "" {
		return fmt.Errorf("last_good 配置为空")
	}

	if err := adaptCaddyfile(server, rollbackConfig); err != nil {
		return fmt.Errorf("回滚 last_good 适配失败: %w", err)
	}
	if err := loadCaddyfile(server, rollbackConfig); err != nil {
		return fmt.Errorf("回滚 last_good 加载失败: %w", err)
	}
	return nil
}
