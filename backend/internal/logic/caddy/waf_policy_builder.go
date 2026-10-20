package caddy

import (
	"fmt"
	wafmodel "logflux/model/waf"
	"strings"
	"unicode"
)

const (
	wafPolicyDefaultAuditRelevantStatus = wafmodel.DefaultAuditRelevantStatus

	wafPolicyCRSTemplateLowFP        = "low_fp"
	wafPolicyCRSTemplateBalanced     = "balanced"
	wafPolicyCRSTemplateHighBlocking = "high_blocking"
	wafPolicyCRSTemplateCustom       = "custom"
	wafPolicyDefaultCRSTemplate      = wafPolicyCRSTemplateLowFP

	wafPolicyMinCRSParanoiaLevel    int64 = 1
	wafPolicyMaxCRSParanoiaLevel    int64 = 4
	wafPolicyMinCRSAnomalyThreshold int64 = 1
	wafPolicyMaxCRSAnomalyThreshold int64 = 20
)

type policyCRSTuningPreset struct {
	ParanoiaLevel            int64
	DetectionParanoiaLevel   int64
	InboundAnomalyThreshold  int64
	OutboundAnomalyThreshold int64
}

// 阈值统一用 CRS 官方推荐值 5/4，强度只调 PL。
//
// 阈值与 PL 是两个正交维度：单条 CRITICAL 规则记 5 分，阈值一旦高于 5，
// 这类攻击就再也无法触发拦截（官方明确超过 5/4 会成片废掉 LFI/RFI/RCE 防护）。
//
// 检测 PL 比阻断 PL 高一档：高 PL 规则先只执行不计分，误报排除干净后再抬高
// 阻断 PL，避免升档直接造成线上拦截。
var wafPolicyCRSTuningPresets = map[string]policyCRSTuningPreset{
	wafPolicyCRSTemplateLowFP: {
		ParanoiaLevel:            1,
		DetectionParanoiaLevel:   1,
		InboundAnomalyThreshold:  5,
		OutboundAnomalyThreshold: 4,
	},
	wafPolicyCRSTemplateBalanced: {
		ParanoiaLevel:            2,
		DetectionParanoiaLevel:   3,
		InboundAnomalyThreshold:  5,
		OutboundAnomalyThreshold: 4,
	},
	wafPolicyCRSTemplateHighBlocking: {
		ParanoiaLevel:            3,
		DetectionParanoiaLevel:   4,
		InboundAnomalyThreshold:  5,
		OutboundAnomalyThreshold: 4,
	},
}

// managedCrsExclusionRules 是随策略下发的 CRS 误报排除规则。
//
// 排除必须在 Include @owasp_crs 之前生效，而策略指令块正好整体位于 CRS 加载
// 之前，因此直接拼在指令末尾即可。
//
// 全部使用 ctl:ruleRemoveById / ctl:ruleRemoveTargetById 做定点排除。不要改成
// ctl:ruleEngine=Off：那会让该路径上的 SQLi、XSS 等全部规则失效。
const managedCrsExclusionRules = `# 管理 API 的请求体是 Caddyfile 原文，会命中 SSRF、协议类规则；
# 只关请求体读取，URI 与 Host 仍参与检测。
SecRule REQUEST_URI "@beginsWith /api/" "id:1000100,phase:1,pass,nolog,noauditlog,ctl:requestBodyAccess=Off"

# Proxmox VE：订阅接口的 localhost 被当作 SSRF（934190）、noVNC 的 package.json
# 命中受限文件（930130）、VNC ticket 的 base64 含 +/= 被当作 SQL 运算符（942120）。
SecRule REQUEST_URI "@rx ^/(api2|novnc|api/v1/guest)/" "id:1000300,phase:1,pass,nolog,noauditlog,ctl:ruleRemoveById=934190,ctl:ruleRemoveById=930130,ctl:ruleRemoveById=942120"

# accept-charset 是弃用但合法的请求头，浏览器与各类客户端都会带。
SecRule REQUEST_HEADERS:Accept-Charset "@rx (?i)charset" "id:1000301,phase:1,pass,nolog,noauditlog,t:none,ctl:ruleRemoveById=920451"

# 第三方埋点 Cookie（PostHog、GA、Sentry 等）的值是 JSON，其中的 $ { } 与逗号
# 会被判成 MongoDB 注入（942290）、MySQL 注释（942200）、SQL 认证绕过（942340）、
# SQL 探测（942370），累计 20 分直接拦截。只排除这几条规则对 Cookie 变量的检测。
SecRule REQUEST_COOKIES_NAMES "@rx ^(ph_|_ga_|_gid|_hj|amplitude|mp_|sentry)" "id:1000302,phase:1,pass,nolog,noauditlog,ctl:ruleRemoveTargetById=942290;REQUEST_COOKIES,ctl:ruleRemoveTargetById=942200;REQUEST_COOKIES,ctl:ruleRemoveTargetById=942340;REQUEST_COOKIES,ctl:ruleRemoveTargetById=942370;REQUEST_COOKIES"`

func normalizePolicyEngineMode(mode string) string {
	normalized := strings.ToLower(strings.TrimSpace(mode))
	if normalized == "" {
		return "on"
	}
	return normalized
}

func validatePolicyEngineMode(mode string) error {
	switch normalizePolicyEngineMode(mode) {
	case "on", "off", "detectiononly":
		return nil
	default:
		return fmt.Errorf("执行模式无效: %s", mode)
	}
}

func normalizePolicyAuditEngine(mode string) string {
	normalized := strings.ToLower(strings.TrimSpace(mode))
	if normalized == "" {
		return "relevantonly"
	}
	return normalized
}

func validatePolicyAuditEngine(mode string) error {
	switch normalizePolicyAuditEngine(mode) {
	case "off", "on", "relevantonly":
		return nil
	default:
		return fmt.Errorf("审计模式无效: %s", mode)
	}
}

func normalizePolicyAuditLogFormat(format string) string {
	normalized := strings.ToLower(strings.TrimSpace(format))
	if normalized == "" {
		return "json"
	}
	return normalized
}

func validatePolicyAuditLogFormat(format string) error {
	switch normalizePolicyAuditLogFormat(format) {
	case "json", "native":
		return nil
	default:
		return fmt.Errorf("审计日志格式无效: %s", format)
	}
}

func normalizePolicyAuditRelevantStatus(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || trimmed == wafmodel.LegacyAuditRelevantStatus {
		return wafPolicyDefaultAuditRelevantStatus
	}
	return trimmed
}

func normalizePolicyRequestBodyLimit(value int64, defaultValue int64) int64 {
	if value <= 0 {
		return defaultValue
	}
	return value
}

func validatePolicyRequestBodyLimit(value int64, field string) error {
	if value <= 0 {
		return fmt.Errorf("%s 必须大于 0", field)
	}
	if value > 1024*1024*1024 {
		return fmt.Errorf("%s 过大", field)
	}
	return nil
}

func normalizePolicyCRSTemplate(template string) string {
	normalized := strings.ToLower(strings.TrimSpace(template))
	if normalized == "" {
		return wafPolicyDefaultCRSTemplate
	}
	return normalized
}

func validatePolicyCRSTemplate(template string) error {
	switch normalizePolicyCRSTemplate(template) {
	case wafPolicyCRSTemplateLowFP, wafPolicyCRSTemplateBalanced, wafPolicyCRSTemplateHighBlocking, wafPolicyCRSTemplateCustom:
		return nil
	default:
		return fmt.Errorf("CRS 模板无效: %s", template)
	}
}

func validatePolicyCRSParanoiaLevel(value int64) error {
	if value < wafPolicyMinCRSParanoiaLevel || value > wafPolicyMaxCRSParanoiaLevel {
		return fmt.Errorf("CRS 偏执级别必须在 %d 到 %d 之间", wafPolicyMinCRSParanoiaLevel, wafPolicyMaxCRSParanoiaLevel)
	}
	return nil
}

func validatePolicyCRSAnomalyThreshold(value int64, field string) error {
	if value < wafPolicyMinCRSAnomalyThreshold || value > wafPolicyMaxCRSAnomalyThreshold {
		return fmt.Errorf("%s 必须在 %d 到 %d 之间", field, wafPolicyMinCRSAnomalyThreshold, wafPolicyMaxCRSAnomalyThreshold)
	}
	return nil
}

func defaultPolicyCRSTuningByTemplate(template string) policyCRSTuningPreset {
	normalized := normalizePolicyCRSTemplate(template)
	if preset, ok := wafPolicyCRSTuningPresets[normalized]; ok {
		return preset
	}
	return wafPolicyCRSTuningPresets[wafPolicyDefaultCRSTemplate]
}

// 按阻断 PL 反推模板。
//
// 阈值不参与：三档预设的阈值统一为 5/4，若仍拿阈值做精确三元组匹配，
// 任何历史数据（如 10/8）都会退化成 custom，而简单模式不接受 custom，
// 用户设置会被静默改写。
func derivePolicyCRSTemplateFromValues(paranoiaLevel int64) string {
	for _, template := range []string{
		wafPolicyCRSTemplateLowFP,
		wafPolicyCRSTemplateBalanced,
		wafPolicyCRSTemplateHighBlocking,
	} {
		if wafPolicyCRSTuningPresets[template].ParanoiaLevel == paranoiaLevel {
			return template
		}
	}
	return wafPolicyCRSTemplateCustom
}

func ensurePolicyCRSTuning(policy *wafmodel.WafPolicy) error {
	if policy == nil {
		return fmt.Errorf("策略为空")
	}

	template := normalizePolicyCRSTemplate(policy.CrsTemplate)
	if err := validatePolicyCRSTemplate(template); err != nil {
		return err
	}

	preset := defaultPolicyCRSTuningByTemplate(template)
	if policy.CrsParanoiaLevel <= 0 {
		policy.CrsParanoiaLevel = preset.ParanoiaLevel
	}
	if policy.CrsInboundAnomalyThreshold <= 0 {
		policy.CrsInboundAnomalyThreshold = preset.InboundAnomalyThreshold
	}
	if policy.CrsOutboundAnomalyThreshold <= 0 {
		policy.CrsOutboundAnomalyThreshold = preset.OutboundAnomalyThreshold
	}

	if err := validatePolicyCRSParanoiaLevel(policy.CrsParanoiaLevel); err != nil {
		return err
	}
	if err := validatePolicyCRSAnomalyThreshold(policy.CrsInboundAnomalyThreshold, "crsInboundAnomalyThreshold"); err != nil {
		return err
	}
	if err := validatePolicyCRSAnomalyThreshold(policy.CrsOutboundAnomalyThreshold, "crsOutboundAnomalyThreshold"); err != nil {
		return err
	}

	// 检测 PL 缺省时取预设值，custom 模板退回等于阻断 PL。
	// 低于阻断 PL 时 CRS 901500 会直接 500 拒绝请求。
	if policy.CrsDetectionParanoiaLevel <= 0 {
		policy.CrsDetectionParanoiaLevel = preset.DetectionParanoiaLevel
		if policy.CrsDetectionParanoiaLevel < policy.CrsParanoiaLevel {
			policy.CrsDetectionParanoiaLevel = policy.CrsParanoiaLevel
		}
	}
	if policy.CrsDetectionParanoiaLevel < policy.CrsParanoiaLevel {
		return fmt.Errorf("CRS 检测级别不能低于阻断级别")
	}
	if err := validatePolicyCRSParanoiaLevel(policy.CrsDetectionParanoiaLevel); err != nil {
		return fmt.Errorf("CRS 检测级别无效: %w", err)
	}

	if template != wafPolicyCRSTemplateCustom && template != derivePolicyCRSTemplateFromValues(policy.CrsParanoiaLevel) {
		template = wafPolicyCRSTemplateCustom
	}
	policy.CrsTemplate = template
	return nil
}

func buildWafPolicyDirectives(policy *wafmodel.WafPolicy) (string, error) {
	if policy == nil {
		return "", fmt.Errorf("策略为空")
	}

	if err := validatePolicyEngineMode(policy.EngineMode); err != nil {
		return "", err
	}
	if err := validatePolicyAuditEngine(policy.AuditEngine); err != nil {
		return "", err
	}
	if err := validatePolicyAuditLogFormat(policy.AuditLogFormat); err != nil {
		return "", err
	}
	if err := validatePolicyRequestBodyLimit(policy.RequestBodyLimit, "requestBodyLimit"); err != nil {
		return "", err
	}
	if err := validatePolicyRequestBodyLimit(policy.RequestBodyNoFilesLimit, "requestBodyNoFilesLimit"); err != nil {
		return "", err
	}
	if err := ensurePolicyCRSTuning(policy); err != nil {
		return "", err
	}

	engineModeMap := map[string]string{
		"on":            "On",
		"off":           "Off",
		"detectiononly": "DetectionOnly",
	}
	auditEngineMap := map[string]string{
		"off":          "Off",
		"on":           "On",
		"relevantonly": "RelevantOnly",
	}
	auditFormatMap := map[string]string{
		"json":   "JSON",
		"native": "Native",
	}

	lines := []string{
		fmt.Sprintf("SecRuleEngine %s", engineModeMap[normalizePolicyEngineMode(policy.EngineMode)]),
		fmt.Sprintf("SecAuditEngine %s", auditEngineMap[normalizePolicyAuditEngine(policy.AuditEngine)]),
		fmt.Sprintf("SecAuditLogFormat %s", auditFormatMap[normalizePolicyAuditLogFormat(policy.AuditLogFormat)]),
		fmt.Sprintf("SecAuditLogRelevantStatus %s", normalizePolicyAuditRelevantStatus(policy.AuditRelevantStatus)),
	}

	if policy.RequestBodyAccess {
		lines = append(lines, "SecRequestBodyAccess On")
	} else {
		lines = append(lines, "SecRequestBodyAccess Off")
	}

	// CRS 4.x 用 blocking/detection 两个变量，裸 tx.paranoia_level 已不再被任何规则读取。
	//
	// detection_paranoia_level 必须 >= blocking_paranoia_level，否则 CRS 901500 直接
	// 500 拒绝请求；该约束已由策略校验保证，这里直接输出。
	// 阈值合并进 900110（官方出站阈值同属此规则），避免与 CRS 900100（异常分值定义）
	// 撞号。
	lines = append(lines,
		fmt.Sprintf("SecRequestBodyLimit %d", policy.RequestBodyLimit),
		fmt.Sprintf("SecRequestBodyNoFilesLimit %d", policy.RequestBodyNoFilesLimit),
		fmt.Sprintf(`SecAction "id:900000,phase:1,pass,nolog,t:none,setvar:tx.blocking_paranoia_level=%d"`, policy.CrsParanoiaLevel),
		fmt.Sprintf(`SecAction "id:900001,phase:1,pass,nolog,t:none,setvar:tx.detection_paranoia_level=%d"`, policy.CrsDetectionParanoiaLevel),
		fmt.Sprintf(`SecAction "id:900110,phase:1,pass,nolog,t:none,setvar:tx.inbound_anomaly_score_threshold=%d,setvar:tx.outbound_anomaly_score_threshold=%d"`, policy.CrsInboundAnomalyThreshold, policy.CrsOutboundAnomalyThreshold),
		"",
		managedCrsExclusionRules,
	)

	return strings.Join(lines, "\n"), nil
}

// applyWafPolicyToCaddyConfig 把策略指令写入配置里所有 Coraza directives 块。
//
// 块内容由 composeManagedCorazaDirectives 统一产出，与首次生成配置时完全一致：
// 三条 CRS Include 必须保留，否则 load_owasp_crs 仍为 true 但一条规则都不会加载。
//
// 逐块替换而不是只改第一块：站点各自内联 coraza_waf 时，只更新第一处会让其余
// 站点继续跑旧策略。从后往前改，前面块的下标不受影响。
func applyWafPolicyToCaddyConfig(caddyConfig, directives string) (string, error) {
	rawConfig := strings.TrimSpace(caddyConfig)
	if rawConfig == "" {
		return "", fmt.Errorf("Caddy 配置为空")
	}
	rawDirectives := strings.TrimSpace(directives)
	if rawDirectives == "" {
		return "", fmt.Errorf("策略指令为空")
	}

	blocks := findCorazaDirectiveBlocks(caddyConfig)
	if len(blocks) == 0 {
		return "", fmt.Errorf("Caddy 配置中未找到 Coraza 指令块")
	}

	renderedLines := strings.Split(
		composeManagedCorazaDirectives(rawDirectives, managedCaddyDefaultWafAuditLog), "\n",
	)

	patched := caddyConfig
	for idx := len(blocks) - 1; idx >= 0; idx-- {
		block := blocks[idx]
		innerIndent := block.Indent + "  "

		lines := make([]string, len(renderedLines))
		for i, line := range renderedLines {
			if strings.TrimSpace(line) == "" {
				lines[i] = ""
				continue
			}
			lines[i] = innerIndent + strings.TrimSpace(line)
		}
		rendered := strings.Join(lines, "\n")

		patched = patched[:block.StartTick+1] + "\n" + rendered + "\n" + block.Indent + patched[block.EndTick:]
	}
	return patched, nil
}

// corazaDirectiveBlock 记录一个 directives 反引号块的位置。
type corazaDirectiveBlock struct {
	StartTick int    // 起始反引号下标
	EndTick   int    // 结束反引号下标
	Indent    string // directives 所在行的缩进
}

// findCorazaDirectiveBlocks 定位配置里所有 directives 反引号块，按出现顺序返回。
func findCorazaDirectiveBlocks(config string) []corazaDirectiveBlock {
	const marker = "directives `"

	blocks := make([]corazaDirectiveBlock, 0, 1)
	cursor := 0
	for {
		tagOffset := strings.Index(config[cursor:], marker)
		if tagOffset < 0 {
			return blocks
		}
		tagIndex := cursor + tagOffset
		startTick := tagIndex + len(marker) - 1

		endOffset := strings.Index(config[startTick+1:], "`")
		if endOffset < 0 {
			// 结束反引号缺失时无法确定边界，交给 /adapt 报错。
			return blocks
		}
		endTick := startTick + 1 + endOffset

		lineStart := strings.LastIndex(config[:tagIndex], "\n") + 1
		blocks = append(blocks, corazaDirectiveBlock{
			StartTick: startTick,
			EndTick:   endTick,
			Indent:    leadingWhitespace(config[lineStart:tagIndex]),
		})
		cursor = endTick + 1
	}
}

func leadingWhitespace(value string) string {
	if value == "" {
		return ""
	}
	var builder strings.Builder
	for _, ch := range value {
		if ch == ' ' || ch == '\t' || unicode.IsSpace(ch) {
			builder.WriteRune(ch)
			continue
		}
		break
	}
	return builder.String()
}
