package caddy

import (
	"context"
	"fmt"
	caddymodel "logflux/model/caddy"
	commonmodel "logflux/model/common"
	wafmodel "logflux/model/waf"
	"strings"

	"logflux/internal/svc"
	"logflux/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

const simpleWafDefaultPolicyName = "default-global-policy"

// simpleWafSiteAddressesKey 是策略 Config 里保存站点选择的键。
//
// 存进 Config 而不是新建绑定表：站点是否真正挂载仍由 Caddyfile 的
// import waf_protect 决定，这里只保存用户的意图，刷新后不再丢。
const simpleWafSiteAddressesKey = "siteAddresses"

type simpleWafConfigService struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logger logx.Logger
}

type simpleWafNormalizedConfig struct {
	Enabled                 bool
	Mode                    string
	Strength                string
	Audit                   string
	RequestBodyAccess       bool
	RequestBodyLimit        int64
	RequestBodyNoFilesLimit int64
	SiteAddresses           []string
}

type simpleWafCandidate struct {
	Server       *caddymodel.CaddyServer
	Policy       *wafmodel.WafPolicy
	Config       string
	Modules      string
	LastGood     string
	LastModules  string
	Directives   string
	Actions      []string
	Snapshot     *wafIntegrationSnapshot
	Normalized   simpleWafNormalizedConfig
	CaddyChanged bool
}

func newSimpleWafConfigService(ctx context.Context, svcCtx *svc.ServiceContext, logger logx.Logger) *simpleWafConfigService {
	return &simpleWafConfigService{
		ctx:    ctx,
		svcCtx: svcCtx,
		logger: logger,
	}
}

// Get 读取 WAF 面板状态；单实例下没有「针对哪个节点」这一层参数。
func (s *simpleWafConfigService) Get(req *types.SimpleWafConfigReq) (*types.SimpleWafConfigResp, error) {
	policy, err := s.findOrCreateDefaultPolicy()
	if err != nil {
		return nil, err
	}

	server, config, _, loadErr := s.loadPreferredConfig()
	if loadErr != nil {
		resp := s.buildResponse(server, policy, nil, nil, nil)
		resp.Message = localizeWafPolicyMessage(loadErr.Error())
		return resp, nil
	}

	snapshot, err := inspectWafIntegration(config)
	if err != nil {
		return nil, err
	}

	directives, err := buildWafPolicyDirectives(policy)
	if err != nil {
		resp := s.buildResponse(server, policy, snapshot, nil, nil)
		resp.Message = localizeWafPolicyMessage(err.Error())
		return resp, nil
	}
	return s.buildResponse(server, policy, snapshot, nil, []string{directives}), nil
}

func (s *simpleWafConfigService) Save(req *types.SimpleWafConfigUpdateReq) error {
	normalized, err := normalizeSimpleWafConfigReq(req)
	if err != nil {
		return err
	}

	policy, err := s.findOrCreateDefaultPolicy()
	if err != nil {
		return err
	}
	if err := applySimpleWafConfigToPolicy(normalized, policy); err != nil {
		return err
	}

	return s.svcCtx.DB.WithContext(s.ctx).Transaction(func(tx *gorm.DB) error {
		if err := ensureSingleDefaultPolicy(tx, policy); err != nil {
			return err
		}
		if err := tx.Save(policy).Error; err != nil {
			return fmt.Errorf("保存简单 WAF 策略失败: %w", err)
		}
		return nil
	})
}

func (s *simpleWafConfigService) Preview(req *types.SimpleWafConfigUpdateReq) (*types.SimpleWafConfigResp, error) {
	candidate, err := s.buildCandidate(req)
	if err != nil {
		return nil, err
	}

	resp := s.buildResponse(candidate.Server, candidate.Policy, candidate.Snapshot, candidate.Actions, []string{candidate.Directives})
	resp.Config = candidate.Config
	resp.Message = "已生成简单 WAF 配置预览"
	return resp, nil
}

func (s *simpleWafConfigService) Apply(req *types.SimpleWafConfigUpdateReq) (*types.SimpleWafConfigResp, error) {
	candidate, err := s.buildCandidate(req)
	if err != nil {
		return nil, err
	}

	// 策略保存与配置落库必须在同一事务：任一步失败整体回滚，避免出现
	// 策略已开而 Caddyfile 未挂载的中间态。
	writer := newCaddyConfigApplyService(s.svcCtx, s.logger)
	if err := writer.Apply(candidate.Server, candidate.Config, candidate.Modules, configWriteOptions{
		HistoryAction:  caddymodel.CaddyHistoryActionSimpleWafApply,
		SnapshotBefore: true,
		BaseHash:       req.BaseHash,
		InTx: func(tx *gorm.DB) error {
			if err := ensureSingleDefaultPolicy(tx, candidate.Policy); err != nil {
				return err
			}
			if err := tx.Save(candidate.Policy).Error; err != nil {
				return fmt.Errorf("保存简单 WAF 策略失败: %w", err)
			}
			return nil
		},
	}); err != nil {
		return nil, err
	}

	// 确保 WAF 审计日志路径已注册为日志源
	if candidate.Normalized.Enabled {
		ensureWafAuditLogSource(s.svcCtx.DB, managedCaddyDefaultWafAuditLog, s.logger)
	}

	resp := s.buildResponse(candidate.Server, candidate.Policy, candidate.Snapshot, candidate.Actions, []string{candidate.Directives})
	resp.Config = candidate.Config
	resp.Message = "简单 WAF 配置已应用"
	return resp, nil
}

func (s *simpleWafConfigService) buildCandidate(req *types.SimpleWafConfigUpdateReq) (*simpleWafCandidate, error) {
	normalized, err := normalizeSimpleWafConfigReq(req)
	if err != nil {
		return nil, err
	}

	// 站点选择来自用户意图，配置文本来自库内现状；两者都可能被并发改过。
	if err := checkBaseHash(s.svcCtx.DB.WithContext(s.ctx), req.BaseHash); err != nil {
		return nil, err
	}

	policy, err := s.findOrCreateDefaultPolicy()
	if err != nil {
		return nil, err
	}
	if err := applySimpleWafConfigToPolicy(normalized, policy); err != nil {
		return nil, err
	}

	directives, err := buildWafPolicyDirectives(policy)
	if err != nil {
		return nil, err
	}

	server, config, modules, err := s.loadPreferredConfig()
	if err != nil {
		if server == nil {
			return nil, err
		}
		config = ""
		modules = emptyModulesJSON
	}

	nextConfig, actions, snapshot, err := buildSimpleWafCandidateConfig(s.svcCtx.DB, config, directives, normalized)
	if err != nil {
		return nil, err
	}

	return &simpleWafCandidate{
		Server:       server,
		Policy:       policy,
		Config:       nextConfig,
		Modules:      modules,
		LastGood:     config,
		LastModules:  modules,
		Directives:   directives,
		Actions:      actions,
		Snapshot:     snapshot,
		Normalized:   normalized,
		CaddyChanged: strings.TrimSpace(config) != strings.TrimSpace(nextConfig),
	}, nil
}

// loadPreferredConfig 取唯一的 Caddy 管理目标与当前生效配置。
func (s *simpleWafConfigService) loadPreferredConfig() (*caddymodel.CaddyServer, string, string, error) {
	server, err := findPreferredCaddyServer(s.svcCtx.DB.WithContext(s.ctx))
	if err != nil {
		return nil, "", emptyModulesJSON, err
	}

	applyService := newCaddyConfigApplyService(s.svcCtx, s.logger)
	config, modules, err := applyService.loadCurrent()
	if err != nil {
		return server, "", emptyModulesJSON, err
	}
	return server, config, modules, nil
}

func (s *simpleWafConfigService) findOrCreateDefaultPolicy() (*wafmodel.WafPolicy, error) {
	db := s.svcCtx.DB.WithContext(s.ctx)
	var policy wafmodel.WafPolicy

	err := db.Where("is_default = ?", true).Order("id asc").First(&policy).Error
	if err == nil {
		return &policy, nil
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("查询默认 WAF 策略失败: %w", err)
	}

	err = db.Where("name = ?", simpleWafDefaultPolicyName).Order("id asc").First(&policy).Error
	if err == nil {
		if !policy.IsDefault {
			policy.IsDefault = true
			if saveErr := db.Save(&policy).Error; saveErr != nil {
				return nil, fmt.Errorf("修正默认 WAF 策略失败: %w", saveErr)
			}
		}
		return &policy, nil
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("查询默认 WAF 策略失败: %w", err)
	}

	policy = wafmodel.WafPolicy{
		Name:                        simpleWafDefaultPolicyName,
		Description:                 "默认简单 WAF 策略",
		Enabled:                     true,
		IsDefault:                   true,
		EngineMode:                  "detectiononly",
		AuditEngine:                 "relevantonly",
		AuditLogFormat:              "json",
		AuditRelevantStatus:         wafPolicyDefaultAuditRelevantStatus,
		RequestBodyAccess:           true,
		RequestBodyLimit:            10 * 1024 * 1024,
		RequestBodyNoFilesLimit:     1024 * 1024,
		CrsTemplate:                 wafPolicyCRSTemplateLowFP,
		CrsParanoiaLevel:            1,
		CrsDetectionParanoiaLevel:   1,
		CrsInboundAnomalyThreshold:  5,
		CrsOutboundAnomalyThreshold: 4,
		Config:                      commonmodel.JSONMap{},
	}
	if err := db.Create(&policy).Error; err != nil {
		return nil, fmt.Errorf("创建默认 WAF 策略失败: %w", err)
	}
	return &policy, nil
}

func (s *simpleWafConfigService) buildResponse(server *caddymodel.CaddyServer, policy *wafmodel.WafPolicy, snapshot *wafIntegrationSnapshot, actions []string, directives []string) *types.SimpleWafConfigResp {
	resp := &types.SimpleWafConfigResp{
		Mode:                    "detectiononly",
		Strength:                wafPolicyCRSTemplateLowFP,
		Audit:                   "relevantonly",
		RequestBodyAccess:       true,
		RequestBodyLimit:        10 * 1024 * 1024,
		RequestBodyNoFilesLimit: 1024 * 1024,
		SiteAddresses:           []string{},
		AvailableSites:          []string{},
		ImportedSites:           []string{},
		Actions:                 actions,
	}
	helper := newWafLogicHelper(s.ctx, s.svcCtx, s.logger)
	resp.CorazaVersion = helper.corazaCurrentVersion()
	resp.CrsVersion = helper.crsCurrentVersion()
	resp.CaddyVersion = helper.caddyCurrentVersion()
	resp.RulesInfo = helper.mountedCRSRulesSummary()
	if policy != nil {
		resp.Mode = normalizePolicyEngineMode(policy.EngineMode)
		resp.Strength = normalizeSimpleWafStrength(policy.CrsTemplate, policy.CrsParanoiaLevel)
		resp.Audit = normalizePolicyAuditEngine(policy.AuditEngine)
		resp.RequestBodyAccess = policy.RequestBodyAccess
		resp.RequestBodyLimit = normalizePolicyRequestBodyLimit(policy.RequestBodyLimit, 10*1024*1024)
		resp.RequestBodyNoFilesLimit = normalizePolicyRequestBodyLimit(policy.RequestBodyNoFilesLimit, 1024*1024)
		resp.Enabled = policy.Enabled && resp.Mode != "off"
	}
	if snapshot != nil {
		resp.Integrated = snapshot.OrderReady && snapshot.SnippetReady && snapshot.DirectiveReady && len(snapshot.ImportedSites) > 0
		resp.AvailableSites = append([]string{}, snapshot.AvailableSites...)
		// Caddyfile 里实际挂了 waf_protect 的站点：前端据此区分"已生效/待应用"。
		resp.ImportedSites = append([]string{}, snapshot.ImportedSites...)
		// 站点选择以策略里保存的为准，Caddyfile 现状只作为"是否已生效"的判据。
		//
		// 不再用 enabled && integrated 改写 enabled：Caddyfile 一旦漂移，UI 会把
		// 用户明明开着的 WAF 显示成关闭，而库里仍是开启状态，两边对不上。
		resp.SiteAddresses = resolveSimpleWafSiteAddresses(policy, snapshot.ImportedSites)
		resp.Message = buildWafIntegrationStatusMessage(snapshot)
	}
	if len(directives) > 0 {
		resp.Directives = strings.TrimSpace(directives[0])
	}
	return resp
}

// resolveSimpleWafSiteAddresses 取策略里保存的站点选择，缺失时回退到 Caddyfile 现状。
func resolveSimpleWafSiteAddresses(policy *wafmodel.WafPolicy, importedSites []string) []string {
	if policy != nil && policy.Config != nil {
		// 同一请求内写入的是 []string，从库里读回的是 []interface{}，两种都要认。
		switch raw := policy.Config[simpleWafSiteAddressesKey].(type) {
		case []string:
			if len(raw) > 0 {
				return raw
			}
		case []interface{}:
			result := make([]string, 0, len(raw))
			for _, item := range raw {
				if text, ok := item.(string); ok && strings.TrimSpace(text) != "" {
					result = append(result, strings.TrimSpace(text))
				}
			}
			if len(result) > 0 {
				return result
			}
		}
	}
	return append([]string{}, importedSites...)
}

func normalizeSimpleWafConfigReq(req *types.SimpleWafConfigUpdateReq) (simpleWafNormalizedConfig, error) {
	if req == nil {
		return simpleWafNormalizedConfig{}, fmt.Errorf("简单 WAF 配置不能为空")
	}

	mode := normalizePolicyEngineMode(req.Mode)
	if strings.TrimSpace(req.Mode) == "" {
		mode = "detectiononly"
	}
	if err := validatePolicyEngineMode(mode); err != nil {
		return simpleWafNormalizedConfig{}, err
	}

	enabled := req.Enabled
	if mode == "off" {
		enabled = false
	}
	if !enabled {
		mode = "off"
	}

	strength := strings.TrimSpace(req.Strength)
	if strength == "" {
		strength = wafPolicyCRSTemplateLowFP
	}
	if err := validatePolicyCRSTemplate(strength); err != nil {
		return simpleWafNormalizedConfig{}, err
	}
	strength = normalizePolicyCRSTemplate(strength)
	if strength == wafPolicyCRSTemplateCustom {
		return simpleWafNormalizedConfig{}, fmt.Errorf("简单模式不支持自定义 CRS 强度")
	}

	audit := normalizePolicyAuditEngine(req.Audit)
	if strings.TrimSpace(req.Audit) == "" {
		audit = "relevantonly"
	}
	if err := validatePolicyAuditEngine(audit); err != nil {
		return simpleWafNormalizedConfig{}, err
	}

	requestBodyLimit := normalizePolicyRequestBodyLimit(req.RequestBodyLimit, 10*1024*1024)
	requestBodyNoFilesLimit := normalizePolicyRequestBodyLimit(req.RequestBodyNoFilesLimit, 1024*1024)
	if err := validatePolicyRequestBodyLimit(requestBodyLimit, "requestBodyLimit"); err != nil {
		return simpleWafNormalizedConfig{}, err
	}
	if err := validatePolicyRequestBodyLimit(requestBodyNoFilesLimit, "requestBodyNoFilesLimit"); err != nil {
		return simpleWafNormalizedConfig{}, err
	}

	siteAddresses := make([]string, 0, len(req.SiteAddresses))
	seen := make(map[string]struct{}, len(req.SiteAddresses))
	for _, item := range req.SiteAddresses {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		siteAddresses = append(siteAddresses, trimmed)
	}

	return simpleWafNormalizedConfig{
		Enabled:                 enabled,
		Mode:                    mode,
		Strength:                strength,
		Audit:                   audit,
		RequestBodyAccess:       req.RequestBodyAccess,
		RequestBodyLimit:        requestBodyLimit,
		RequestBodyNoFilesLimit: requestBodyNoFilesLimit,
		SiteAddresses:           siteAddresses,
	}, nil
}

func applySimpleWafConfigToPolicy(config simpleWafNormalizedConfig, policy *wafmodel.WafPolicy) error {
	if policy == nil {
		return fmt.Errorf("默认 WAF 策略不存在")
	}

	preset := defaultPolicyCRSTuningByTemplate(config.Strength)
	if strings.TrimSpace(policy.Name) == "" {
		policy.Name = simpleWafDefaultPolicyName
	}
	if strings.TrimSpace(policy.Description) == "" {
		policy.Description = "默认简单 WAF 策略"
	}
	policy.Enabled = config.Enabled
	policy.IsDefault = true
	policy.EngineMode = config.Mode
	policy.AuditEngine = config.Audit
	policy.AuditLogFormat = "json"
	policy.AuditRelevantStatus = wafPolicyDefaultAuditRelevantStatus
	policy.RequestBodyAccess = config.RequestBodyAccess
	policy.RequestBodyLimit = config.RequestBodyLimit
	policy.RequestBodyNoFilesLimit = config.RequestBodyNoFilesLimit
	policy.CrsTemplate = config.Strength
	policy.CrsParanoiaLevel = preset.ParanoiaLevel
	policy.CrsDetectionParanoiaLevel = preset.DetectionParanoiaLevel
	policy.CrsInboundAnomalyThreshold = preset.InboundAnomalyThreshold
	policy.CrsOutboundAnomalyThreshold = preset.OutboundAnomalyThreshold
	if policy.Config == nil {
		policy.Config = commonmodel.JSONMap{}
	}
	// 直接存 []string：JSONMap 落库时统一走 json.Marshal，读回时由解析器还原。
	policy.Config[simpleWafSiteAddressesKey] = config.SiteAddresses

	return ensurePolicyCRSTuning(policy)
}

func normalizeSimpleWafStrength(template string, paranoiaLevel int64) string {
	normalized := normalizePolicyCRSTemplate(template)
	switch normalized {
	case wafPolicyCRSTemplateLowFP, wafPolicyCRSTemplateBalanced, wafPolicyCRSTemplateHighBlocking:
		return normalized
	}
	derived := derivePolicyCRSTemplateFromValues(paranoiaLevel)
	switch derived {
	case wafPolicyCRSTemplateLowFP, wafPolicyCRSTemplateBalanced, wafPolicyCRSTemplateHighBlocking:
		return derived
	default:
		return wafPolicyCRSTemplateBalanced
	}
}

func buildSimpleWafCandidateConfig(db *gorm.DB, currentConfig, directives string, config simpleWafNormalizedConfig) (string, []string, *wafIntegrationSnapshot, error) {
	if strings.TrimSpace(directives) == "" {
		return "", nil, nil, fmt.Errorf("WAF 指令为空")
	}

	actions := make([]string, 0)
	if strings.TrimSpace(currentConfig) == "" {
		rendered, err := renderManagedCaddyfile(managedCaddyfileOptions{
			SiteAddress:        managedCaddyDefaultSiteAddress,
			Backend:            managedCaddyDefaultBackend,
			FrontendRoot:       managedCaddyDefaultFrontend,
			AccessLogPath:      managedCaddyDefaultAccessLog,
			WafAuditLog:        managedCaddyDefaultWafAuditLog,
			WafEnabled:         config.Enabled,
			ForwardAuthEnabled: isIPRegionEnabled(db),
			Directives:         directives,
		})
		if err != nil {
			return "", nil, nil, err
		}
		snapshot, err := inspectWafIntegration(rendered)
		if err != nil {
			return "", nil, nil, err
		}
		if config.Enabled {
			actions = append(actions, "生成默认 Caddyfile 并启用 waf_protect")
		} else {
			actions = append(actions, "生成默认 Caddyfile 并保持 WAF 关闭")
		}
		return rendered, actions, snapshot, nil
	}

	nextConfig := currentConfig
	if config.Enabled {
		var changed bool
		var err error
		nextConfig, changed, err = ensureCorazaOrder(nextConfig)
		if err != nil {
			return "", nil, nil, err
		}
		if changed {
			actions = append(actions, "注入全局 order coraza_waf first")
		}

		nextConfig, changed, err = ensureWafProtectSnippet(nextConfig)
		if err != nil {
			return "", nil, nil, err
		}
		if changed {
			actions = append(actions, "注入 waf_protect 统一片段")
		}

		updatedConfig, err := applyWafPolicyToCaddyConfig(nextConfig, directives)
		if err != nil {
			return "", nil, nil, err
		}
		if strings.TrimSpace(updatedConfig) != strings.TrimSpace(nextConfig) {
			actions = append(actions, "更新 Coraza 策略指令")
		}
		nextConfig = updatedConfig

		snapshot, err := inspectWafIntegration(nextConfig)
		if err != nil {
			return "", nil, nil, err
		}
		targetSites := resolveSimpleWafTargetSites(snapshot.AvailableSites, snapshot.ImportedSites, config.SiteAddresses, true)
		if len(targetSites) == 0 {
			return "", nil, nil, fmt.Errorf("未识别到可接入的站点")
		}

		for _, siteAddress := range targetSites {
			nextConfig, changed, err = ensureSiteImport(nextConfig, siteAddress)
			if err != nil {
				return "", nil, nil, err
			}
			if changed {
				actions = append(actions, fmt.Sprintf("为站点 %s 挂载 waf_protect", siteAddress))
			}
		}

		snapshot, err = inspectWafIntegration(nextConfig)
		if err != nil {
			return "", nil, nil, err
		}
		if len(actions) == 0 {
			actions = append(actions, "WAF 配置无变更")
		}
		return nextConfig, actions, snapshot, nil
	}

	snapshot, err := inspectWafIntegration(nextConfig)
	if err != nil {
		return "", nil, nil, err
	}
	targetSites := resolveSimpleWafTargetSites(snapshot.AvailableSites, snapshot.ImportedSites, config.SiteAddresses, false)
	for _, siteAddress := range targetSites {
		var changed bool
		nextConfig, changed, err = removeSiteImport(nextConfig, siteAddress)
		if err != nil {
			return "", nil, nil, err
		}
		if changed {
			actions = append(actions, fmt.Sprintf("取消站点 %s 的 waf_protect 挂载", siteAddress))
		}
	}
	if snapshot.DirectiveReady {
		if updatedConfig, updateErr := applyWafPolicyToCaddyConfig(nextConfig, directives); updateErr == nil {
			if strings.TrimSpace(updatedConfig) != strings.TrimSpace(nextConfig) {
				actions = append(actions, "更新 Coraza 策略指令为 Off")
			}
			nextConfig = updatedConfig
		}
	}
	snapshot, err = inspectWafIntegration(nextConfig)
	if err != nil {
		return "", nil, nil, err
	}
	if len(actions) == 0 {
		actions = append(actions, "WAF 已保持关闭")
	}
	return nextConfig, actions, snapshot, nil
}

func resolveSimpleWafTargetSites(availableSites, importedSites, requestedSites []string, enabled bool) []string {
	availableSet := make(map[string]struct{}, len(availableSites))
	for _, site := range availableSites {
		trimmed := strings.TrimSpace(site)
		if trimmed != "" {
			availableSet[trimmed] = struct{}{}
		}
	}

	pickKnown := func(source []string) []string {
		result := make([]string, 0, len(source))
		seen := make(map[string]struct{}, len(source))
		for _, item := range source {
			trimmed := strings.TrimSpace(item)
			if trimmed == "" {
				continue
			}
			if _, ok := availableSet[trimmed]; len(availableSet) > 0 && !ok {
				continue
			}
			if _, ok := seen[trimmed]; ok {
				continue
			}
			seen[trimmed] = struct{}{}
			result = append(result, trimmed)
		}
		return result
	}

	if len(requestedSites) > 0 {
		return pickKnown(requestedSites)
	}
	if enabled {
		return pickKnown(availableSites)
	}
	if len(importedSites) > 0 {
		return pickKnown(importedSites)
	}
	return pickKnown(availableSites)
}
