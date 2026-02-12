package caddy

import (
	"fmt"
	"strings"
	"unicode"

	"logflux/model"
)

const (
	wafPolicyDefaultAuditRelevantStatus = "^(?:5|4(?!04))"

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
	InboundAnomalyThreshold  int64
	OutboundAnomalyThreshold int64
}

var wafPolicyCRSTuningPresets = map[string]policyCRSTuningPreset{
	wafPolicyCRSTemplateLowFP: {
		ParanoiaLevel:            1,
		InboundAnomalyThreshold:  10,
		OutboundAnomalyThreshold: 8,
	},
	wafPolicyCRSTemplateBalanced: {
		ParanoiaLevel:            2,
		InboundAnomalyThreshold:  5,
		OutboundAnomalyThreshold: 4,
	},
	wafPolicyCRSTemplateHighBlocking: {
		ParanoiaLevel:            3,
		InboundAnomalyThreshold:  3,
		OutboundAnomalyThreshold: 2,
	},
}

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
		return fmt.Errorf("invalid engine mode: %s", mode)
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
		return fmt.Errorf("invalid audit engine: %s", mode)
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
		return fmt.Errorf("invalid audit log format: %s", format)
	}
}

func normalizePolicyAuditRelevantStatus(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
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
		return fmt.Errorf("%s must be greater than 0", field)
	}
	if value > 1024*1024*1024 {
		return fmt.Errorf("%s is too large", field)
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
		return fmt.Errorf("invalid crs template: %s", template)
	}
}

func validatePolicyCRSParanoiaLevel(value int64) error {
	if value < wafPolicyMinCRSParanoiaLevel || value > wafPolicyMaxCRSParanoiaLevel {
		return fmt.Errorf("crsParanoiaLevel must be between %d and %d", wafPolicyMinCRSParanoiaLevel, wafPolicyMaxCRSParanoiaLevel)
	}
	return nil
}

func validatePolicyCRSAnomalyThreshold(value int64, field string) error {
	if value < wafPolicyMinCRSAnomalyThreshold || value > wafPolicyMaxCRSAnomalyThreshold {
		return fmt.Errorf("%s must be between %d and %d", field, wafPolicyMinCRSAnomalyThreshold, wafPolicyMaxCRSAnomalyThreshold)
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

func derivePolicyCRSTemplateFromValues(paranoiaLevel, inboundThreshold, outboundThreshold int64) string {
	for _, template := range []string{
		wafPolicyCRSTemplateLowFP,
		wafPolicyCRSTemplateBalanced,
		wafPolicyCRSTemplateHighBlocking,
	} {
		preset := wafPolicyCRSTuningPresets[template]
		if preset.ParanoiaLevel == paranoiaLevel &&
			preset.InboundAnomalyThreshold == inboundThreshold &&
			preset.OutboundAnomalyThreshold == outboundThreshold {
			return template
		}
	}
	return wafPolicyCRSTemplateCustom
}

func ensurePolicyCRSTuning(policy *model.WafPolicy) error {
	if policy == nil {
		return fmt.Errorf("policy is nil")
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

	derivedTemplate := derivePolicyCRSTemplateFromValues(
		policy.CrsParanoiaLevel,
		policy.CrsInboundAnomalyThreshold,
		policy.CrsOutboundAnomalyThreshold,
	)

	if template != wafPolicyCRSTemplateCustom && template != derivedTemplate {
		template = wafPolicyCRSTemplateCustom
	}
	policy.CrsTemplate = template
	return nil
}

func buildWafPolicyDirectives(policy *model.WafPolicy) (string, error) {
	if policy == nil {
		return "", fmt.Errorf("policy is nil")
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

	lines = append(lines,
		fmt.Sprintf("SecRequestBodyLimit %d", policy.RequestBodyLimit),
		fmt.Sprintf("SecRequestBodyNoFilesLimit %d", policy.RequestBodyNoFilesLimit),
		fmt.Sprintf(`SecAction "id:900000,phase:1,pass,nolog,t:none,setvar:tx.paranoia_level=%d"`, policy.CrsParanoiaLevel),
		fmt.Sprintf(`SecAction "id:900110,phase:1,pass,nolog,t:none,setvar:tx.inbound_anomaly_score_threshold=%d"`, policy.CrsInboundAnomalyThreshold),
		fmt.Sprintf(`SecAction "id:900100,phase:1,pass,nolog,t:none,setvar:tx.outbound_anomaly_score_threshold=%d"`, policy.CrsOutboundAnomalyThreshold),
	)

	return strings.Join(lines, "\n"), nil
}

func applyWafPolicyToCaddyConfig(caddyConfig, directives string) (string, error) {
	rawConfig := strings.TrimSpace(caddyConfig)
	if rawConfig == "" {
		return "", fmt.Errorf("caddy config is empty")
	}
	rawDirectives := strings.TrimSpace(directives)
	if rawDirectives == "" {
		return "", fmt.Errorf("policy directives is empty")
	}

	directiveTagIndex := strings.Index(caddyConfig, "directives `")
	if directiveTagIndex < 0 {
		return "", fmt.Errorf("coraza directives block not found in caddy config")
	}

	firstTickOffset := strings.Index(caddyConfig[directiveTagIndex:], "`")
	if firstTickOffset < 0 {
		return "", fmt.Errorf("coraza directives start tick not found")
	}
	startTickIndex := directiveTagIndex + firstTickOffset

	secondTickOffset := strings.Index(caddyConfig[startTickIndex+1:], "`")
	if secondTickOffset < 0 {
		return "", fmt.Errorf("coraza directives end tick not found")
	}
	endTickIndex := startTickIndex + 1 + secondTickOffset

	lineStart := strings.LastIndex(caddyConfig[:directiveTagIndex], "\n") + 1
	directiveLine := caddyConfig[lineStart:directiveTagIndex]
	indent := leadingWhitespace(directiveLine)
	innerIndent := indent + "  "

	lines := strings.Split(rawDirectives, "\n")
	for i := range lines {
		lines[i] = innerIndent + strings.TrimSpace(lines[i])
	}
	rendered := strings.Join(lines, "\n")

	patched := caddyConfig[:startTickIndex+1] + "\n" + rendered + "\n" + indent + caddyConfig[endTickIndex:]
	return patched, nil
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
