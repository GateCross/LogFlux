package caddy

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"logflux/model"
)

var (
	regexDirectiveEngineMode              = regexp.MustCompile(`(?im)^\s*SecRuleEngine\s+([^\s#]+)`)
	regexDirectiveAuditEngine             = regexp.MustCompile(`(?im)^\s*SecAuditEngine\s+([^\s#]+)`)
	regexDirectiveAuditLogFormat          = regexp.MustCompile(`(?im)^\s*SecAuditLogFormat\s+([^\s#]+)`)
	regexDirectiveAuditRelevantStatus     = regexp.MustCompile(`(?im)^\s*SecAuditLogRelevantStatus\s+(.+)$`)
	regexDirectiveRequestBodyAccess       = regexp.MustCompile(`(?im)^\s*SecRequestBodyAccess\s+([^\s#]+)`)
	regexDirectiveRequestBodyLimit        = regexp.MustCompile(`(?im)^\s*SecRequestBodyLimit\s+([^\s#]+)`)
	regexDirectiveRequestBodyNoFilesLimit = regexp.MustCompile(`(?im)^\s*SecRequestBodyNoFilesLimit\s+([^\s#]+)`)
	regexDirectiveParanoiaLevel           = regexp.MustCompile(`(?i)setvar:tx\.paranoia_level=([0-9]+)`)
	regexDirectiveInboundThreshold        = regexp.MustCompile(`(?i)setvar:tx\.inbound_anomaly_score_threshold=([0-9]+)`)
	regexDirectiveOutboundThreshold       = regexp.MustCompile(`(?i)setvar:tx\.outbound_anomaly_score_threshold=([0-9]+)`)
)

func buildWafPolicyRevisionChangeSummary(current *model.WafPolicyRevision, previous *model.WafPolicyRevision) string {
	if current == nil {
		return "-"
	}

	baseSummary := summarizeWafPolicyRevisionMessage(current.Message)
	changeParts := make([]string, 0, 6)
	if previous != nil {
		directiveChanges := diffWafPolicyDirectiveFields(previous.DirectivesSnapshot, current.DirectivesSnapshot)
		if len(directiveChanges) > 0 {
			changeParts = append(changeParts, directiveChanges...)
		}
		if !reflect.DeepEqual(previous.ConfigSnapshot, current.ConfigSnapshot) {
			changeParts = append(changeParts, "扩展配置")
		}
		if strings.TrimSpace(previous.Status) != strings.TrimSpace(current.Status) {
			changeParts = append(changeParts, "版本状态")
		}
	}

	if baseSummary == "" && len(changeParts) == 0 {
		return "-"
	}
	if len(changeParts) == 0 {
		return baseSummary
	}
	if baseSummary == "" {
		return strings.Join(changeParts, "、")
	}
	return fmt.Sprintf("%s（%s）", baseSummary, strings.Join(deduplicateAuditFields(changeParts), "、"))
}

func summarizeWafPolicyRevisionMessage(message string) string {
	raw := strings.TrimSpace(message)
	if raw == "" {
		return ""
	}

	lower := strings.ToLower(raw)
	switch lower {
	case "create policy":
		return "创建策略草稿"
	case "update policy":
		return "更新策略草稿"
	case "publish policy":
		return "发布策略"
	case "rollback policy":
		return "回滚策略"
	case "init default policy":
		return "初始化默认策略"
	default:
		return raw
	}
}

func diffWafPolicyDirectiveFields(previous, current string) []string {
	prevValues := extractWafDirectiveAuditValues(previous)
	currValues := extractWafDirectiveAuditValues(current)
	changed := make([]string, 0, len(currValues))

	fieldLabels := []struct {
		key   string
		label string
	}{
		{key: "engine_mode", label: "引擎模式"},
		{key: "audit_engine", label: "审计模式"},
		{key: "audit_log_format", label: "审计日志格式"},
		{key: "audit_relevant_status", label: "审计匹配状态"},
		{key: "request_body_access", label: "请求体访问开关"},
		{key: "request_body_limit", label: "请求体大小限制"},
		{key: "request_body_no_files_limit", label: "无文件请求体大小限制"},
		{key: "crs_paranoia_level", label: "CRS PL"},
		{key: "crs_inbound_threshold", label: "CRS 入站阈值"},
		{key: "crs_outbound_threshold", label: "CRS 出站阈值"},
	}

	for _, field := range fieldLabels {
		prevValue := strings.TrimSpace(prevValues[field.key])
		currValue := strings.TrimSpace(currValues[field.key])
		if prevValue == currValue {
			continue
		}
		if prevValue == "" && currValue == "" {
			continue
		}
		changed = append(changed, field.label)
	}

	if strings.TrimSpace(previous) != strings.TrimSpace(current) && len(changed) == 0 {
		changed = append(changed, "指令内容")
	}
	return changed
}

func extractWafDirectiveAuditValues(directives string) map[string]string {
	text := strings.TrimSpace(directives)
	values := make(map[string]string, 10)
	if text == "" {
		return values
	}

	extract := func(key string, pattern *regexp.Regexp) {
		matches := pattern.FindStringSubmatch(text)
		if len(matches) != 2 {
			return
		}
		values[key] = strings.TrimSpace(matches[1])
	}

	extract("engine_mode", regexDirectiveEngineMode)
	extract("audit_engine", regexDirectiveAuditEngine)
	extract("audit_log_format", regexDirectiveAuditLogFormat)
	extract("audit_relevant_status", regexDirectiveAuditRelevantStatus)
	extract("request_body_access", regexDirectiveRequestBodyAccess)
	extract("request_body_limit", regexDirectiveRequestBodyLimit)
	extract("request_body_no_files_limit", regexDirectiveRequestBodyNoFilesLimit)
	extract("crs_paranoia_level", regexDirectiveParanoiaLevel)
	extract("crs_inbound_threshold", regexDirectiveInboundThreshold)
	extract("crs_outbound_threshold", regexDirectiveOutboundThreshold)

	return values
}

func deduplicateAuditFields(fields []string) []string {
	if len(fields) == 0 {
		return fields
	}
	seen := make(map[string]struct{}, len(fields))
	result := make([]string, 0, len(fields))
	for _, field := range fields {
		normalized := strings.TrimSpace(field)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		result = append(result, normalized)
	}
	return result
}
