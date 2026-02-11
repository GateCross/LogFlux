package caddy

import (
	"fmt"
	"strings"
	"unicode"

	"logflux/model"
)

const (
	wafPolicyDefaultAuditRelevantStatus = "^(?:5|4(?!04))"
)

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
