package caddy

import (
	"fmt"
	"strings"

	"logflux/model"

	"gorm.io/gorm"
)

const (
	wafPolicyScopeTypeGlobal = "global"
	wafPolicyScopeTypeSite   = "site"
	wafPolicyScopeTypeRoute  = "route"

	wafPolicyRemoveTypeID  = "id"
	wafPolicyRemoveTypeTag = "tag"

	wafPolicyBindingDefaultPriority int64 = 100
	wafPolicyBindingMinPriority     int64 = 1
	wafPolicyBindingMaxPriority     int64 = 1000
)

func normalizePolicyScopeType(scopeType string) string {
	normalized := strings.ToLower(strings.TrimSpace(scopeType))
	if normalized == "" {
		return wafPolicyScopeTypeGlobal
	}
	return normalized
}

func validatePolicyScopeType(scopeType string) error {
	switch normalizePolicyScopeType(scopeType) {
	case wafPolicyScopeTypeGlobal, wafPolicyScopeTypeSite, wafPolicyScopeTypeRoute:
		return nil
	default:
		return fmt.Errorf("invalid policy scope type: %s", scopeType)
	}
}

func normalizePolicyRemoveType(removeType string) string {
	normalized := strings.ToLower(strings.TrimSpace(removeType))
	if normalized == "" {
		return wafPolicyRemoveTypeID
	}
	return normalized
}

func validatePolicyRemoveType(removeType string) error {
	switch normalizePolicyRemoveType(removeType) {
	case wafPolicyRemoveTypeID, wafPolicyRemoveTypeTag:
		return nil
	default:
		return fmt.Errorf("invalid policy remove type: %s", removeType)
	}
}

func normalizePolicyHTTPMethod(method string) string {
	return strings.ToUpper(strings.TrimSpace(method))
}

func validatePolicyHTTPMethod(method string) error {
	switch normalizePolicyHTTPMethod(method) {
	case "", "GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS":
		return nil
	default:
		return fmt.Errorf("invalid policy method: %s", method)
	}
}

func normalizePolicyScopePath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return ""
	}
	if strings.HasPrefix(trimmed, "/") {
		return trimmed
	}
	return "/" + trimmed
}

func normalizePolicyScopeHost(host string) string {
	return strings.ToLower(strings.TrimSpace(host))
}

func normalizePolicyBindingPriority(priority int64) int64 {
	if priority <= 0 {
		return wafPolicyBindingDefaultPriority
	}
	return priority
}

func validatePolicyBindingPriority(priority int64) error {
	if priority < wafPolicyBindingMinPriority || priority > wafPolicyBindingMaxPriority {
		return fmt.Errorf("binding priority must be between %d and %d", wafPolicyBindingMinPriority, wafPolicyBindingMaxPriority)
	}
	return nil
}

func normalizeAndValidateExclusionScopeFields(scopeType, host, path, method string) (string, string, string, string, error) {
	normalizedScopeType := normalizePolicyScopeType(scopeType)
	if err := validatePolicyScopeType(normalizedScopeType); err != nil {
		return "", "", "", "", err
	}

	normalizedHost := normalizePolicyScopeHost(host)
	normalizedPath := normalizePolicyScopePath(path)
	normalizedMethod := normalizePolicyHTTPMethod(method)

	if err := validatePolicyHTTPMethod(normalizedMethod); err != nil {
		return "", "", "", "", err
	}

	switch normalizedScopeType {
	case wafPolicyScopeTypeGlobal:
		normalizedHost = ""
		normalizedPath = ""
		normalizedMethod = ""
	case wafPolicyScopeTypeSite:
		if normalizedHost == "" {
			return "", "", "", "", fmt.Errorf("site scope requires host")
		}
		normalizedPath = ""
		normalizedMethod = ""
	case wafPolicyScopeTypeRoute:
		if normalizedPath == "" {
			return "", "", "", "", fmt.Errorf("route scope requires path")
		}
	}

	return normalizedScopeType, normalizedHost, normalizedPath, normalizedMethod, nil
}

func normalizeAndValidateBindingScopeFields(scopeType, host, path, method string) (string, string, string, string, error) {
	normalizedScopeType := normalizePolicyScopeType(scopeType)
	if err := validatePolicyScopeType(normalizedScopeType); err != nil {
		return "", "", "", "", err
	}

	normalizedHost := normalizePolicyScopeHost(host)
	normalizedPath := normalizePolicyScopePath(path)
	normalizedMethod := normalizePolicyHTTPMethod(method)

	if err := validatePolicyHTTPMethod(normalizedMethod); err != nil {
		return "", "", "", "", err
	}

	switch normalizedScopeType {
	case wafPolicyScopeTypeGlobal:
		normalizedHost = ""
		normalizedPath = ""
		normalizedMethod = ""
	case wafPolicyScopeTypeSite:
		if normalizedHost == "" {
			return "", "", "", "", fmt.Errorf("site scope requires host")
		}
		normalizedPath = ""
		normalizedMethod = ""
	case wafPolicyScopeTypeRoute:
		if normalizedPath == "" {
			return "", "", "", "", fmt.Errorf("route scope requires path")
		}
	}

	return normalizedScopeType, normalizedHost, normalizedPath, normalizedMethod, nil
}

func validatePolicyIDExists(db *gorm.DB, policyID uint) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	if policyID == 0 {
		return fmt.Errorf("policy id is required")
	}
	var count int64
	if err := db.Model(&model.WafPolicy{}).Where("id = ?", policyID).Count(&count).Error; err != nil {
		return fmt.Errorf("query policy failed: %w", err)
	}
	if count == 0 {
		return fmt.Errorf("policy not found")
	}
	return nil
}

func buildWafRuleExclusionDirectives(exclusions []model.WafRuleExclusion) (string, error) {
	if len(exclusions) == 0 {
		return "", nil
	}

	ruleID := int64(990000)
	lines := make([]string, 0, len(exclusions)*3)

	for _, exclusion := range exclusions {
		if !exclusion.Enabled {
			continue
		}

		removeType := normalizePolicyRemoveType(exclusion.RemoveType)
		if err := validatePolicyRemoveType(removeType); err != nil {
			return "", err
		}
		removeValue := strings.TrimSpace(exclusion.RemoveValue)
		if removeValue == "" {
			return "", fmt.Errorf("remove value is required")
		}

		scopeType, host, path, method, err := normalizeAndValidateExclusionScopeFields(
			exclusion.ScopeType,
			exclusion.Host,
			exclusion.Path,
			exclusion.Method,
		)
		if err != nil {
			return "", err
		}

		switch scopeType {
		case wafPolicyScopeTypeGlobal:
			if removeType == wafPolicyRemoveTypeID {
				lines = append(lines, fmt.Sprintf("SecRuleRemoveById %s", removeValue))
			} else {
				lines = append(lines, fmt.Sprintf("SecRuleRemoveByTag %s", removeValue))
			}
		case wafPolicyScopeTypeSite, wafPolicyScopeTypeRoute:
			scopedLines, err := buildScopedWafRuleExclusionDirectives(ruleID, host, path, method, removeType, removeValue)
			if err != nil {
				return "", err
			}
			ruleID += 10
			lines = append(lines, scopedLines...)
		default:
			return "", fmt.Errorf("invalid policy scope type: %s", scopeType)
		}
	}

	return strings.TrimSpace(strings.Join(lines, "\n")), nil
}

func buildScopedWafRuleExclusionDirectives(
	ruleID int64,
	host, path, method, removeType, removeValue string,
) ([]string, error) {
	type matcher struct {
		Variable string
		Operator string
		Value    string
	}

	matchers := make([]matcher, 0, 3)
	if host != "" {
		matchers = append(matchers, matcher{Variable: "REQUEST_HEADERS:Host", Operator: "@streq", Value: host})
	}
	if path != "" {
		matchers = append(matchers, matcher{Variable: "REQUEST_URI", Operator: "@beginsWith", Value: path})
	}
	if method != "" {
		matchers = append(matchers, matcher{Variable: "REQUEST_METHOD", Operator: "@streq", Value: method})
	}

	if len(matchers) == 0 {
		return nil, fmt.Errorf("scoped exclusion matchers is empty")
	}

	controlAction := "ctl:ruleRemoveById=" + removeValue
	if removeType == wafPolicyRemoveTypeTag {
		controlAction = "ctl:ruleRemoveByTag=" + removeValue
	}

	lines := make([]string, 0, len(matchers))
	for idx, currentMatcher := range matchers {
		actions := []string{}
		if idx == 0 {
			actions = append(actions, fmt.Sprintf("id:%d", ruleID), "phase:1", "pass", "nolog", "t:none")
		}
		if idx < len(matchers)-1 {
			actions = append(actions, "chain")
		} else {
			actions = append(actions, controlAction)
		}
		lines = append(lines, fmt.Sprintf(
			`SecRule %s "%s %s" "%s"`,
			currentMatcher.Variable,
			currentMatcher.Operator,
			strings.ReplaceAll(currentMatcher.Value, `"`, `\"`),
			strings.Join(actions, ","),
		))
	}

	return lines, nil
}

func buildPolicyDirectivesWithExclusions(db *gorm.DB, policy *model.WafPolicy) (string, error) {
	baseDirectives, err := buildWafPolicyDirectives(policy)
	if err != nil {
		return "", err
	}
	if db == nil || policy == nil || policy.ID == 0 {
		return baseDirectives, nil
	}

	var exclusions []model.WafRuleExclusion
	if err := db.Where("policy_id = ? AND enabled = ?", policy.ID, true).Order("id asc").Find(&exclusions).Error; err != nil {
		return "", fmt.Errorf("query policy exclusions failed: %w", err)
	}

	exclusionDirectives, err := buildWafRuleExclusionDirectives(exclusions)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(exclusionDirectives) == "" {
		return baseDirectives, nil
	}
	return strings.TrimSpace(baseDirectives) + "\n" + strings.TrimSpace(exclusionDirectives), nil
}

func normalizeBindingScopeKey(scopeType, host, path, method string, priority int64) string {
	return fmt.Sprintf("%s|%s|%s|%s|%d", scopeType, host, path, method, priority)
}

func validatePolicyBindingConflict(tx *gorm.DB, candidate *model.WafPolicyBinding) error {
	if tx == nil || candidate == nil {
		return fmt.Errorf("invalid policy binding context")
	}

	var count int64
	query := tx.Model(&model.WafPolicyBinding{}).
		Where("enabled = ? AND scope_type = ? AND host = ? AND path = ? AND method = ? AND priority = ?",
			true, candidate.ScopeType, candidate.Host, candidate.Path, candidate.Method, candidate.Priority)
	if candidate.ID > 0 {
		query = query.Where("id <> ?", candidate.ID)
	}
	if err := query.Count(&count).Error; err != nil {
		return fmt.Errorf("query policy binding conflict failed: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("policy binding conflict detected for scope=%s host=%s path=%s method=%s priority=%d",
			candidate.ScopeType, candidate.Host, candidate.Path, candidate.Method, candidate.Priority)
	}
	return nil
}

func ensureNoPolicyBindingConflicts(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}

	type bindingGroup struct {
		ScopeType string
		Host      string
		Path      string
		Method    string
		Priority  int64
		Count     int64
	}

	var conflicts []bindingGroup
	if err := db.Model(&model.WafPolicyBinding{}).
		Select("scope_type, host, path, method, priority, COUNT(*) as count").
		Where("enabled = ?", true).
		Group("scope_type, host, path, method, priority").
		Having("COUNT(*) > 1").
		Order("count desc, priority asc").
		Limit(1).
		Scan(&conflicts).Error; err != nil {
		return fmt.Errorf("query policy binding conflicts failed: %w", err)
	}
	if len(conflicts) == 0 {
		return nil
	}

	conflict := conflicts[0]
	return fmt.Errorf(
		"policy binding conflicts found: scope=%s host=%s path=%s method=%s priority=%d count=%d",
		conflict.ScopeType,
		conflict.Host,
		conflict.Path,
		conflict.Method,
		conflict.Priority,
		conflict.Count,
	)
}
