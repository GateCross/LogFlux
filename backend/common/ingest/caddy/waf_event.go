package caddy

import (
	"encoding/json"
	"strings"
	"time"

	"logflux/common/ingest/shared"
	caddymodel "logflux/model/caddy"
	wafmodel "logflux/model/waf"
)

// 审计来源标记，与 parseCorazaAuditJSON 写入 ExtraData 的值保持一致
const wafExtraSourceTag = `"source":"waf"`

// isWafAuditEntry 判断访问日志是否来自 Coraza 审计日志。
func isWafAuditEntry(entry *caddymodel.CaddyLog) bool {
	if entry == nil {
		return false
	}
	return strings.Contains(entry.ExtraData, wafExtraSourceTag)
}

type wafExtraMeta struct {
	Source          string `json:"source"`
	TransactionID   string `json:"transaction_id"`
	Interrupted     bool   `json:"interrupted"`
	HighestSeverity string `json:"highest_severity"`
	MatchedRules    []struct {
		ID       any    `json:"id"`
		Msg      string `json:"msg"`
		Severity any    `json:"severity"`
		Message  string `json:"message"`
	} `json:"matched_rules"`
}

// WafEventsFromCaddyLog 把审计来源的访问日志归一化为 WAF 事件，一次规则命中一行；
// 无命中规则的事务产出一条 RuleID 为空的事件。输入不是审计来源时返回 nil。
func WafEventsFromCaddyLog(entry *caddymodel.CaddyLog) []*wafmodel.WafEvent {
	if !isWafAuditEntry(entry) {
		return nil
	}

	var meta wafExtraMeta
	if err := json.Unmarshal([]byte(entry.ExtraData), &meta); err != nil {
		return nil
	}

	eventTime := entry.LogTime
	if eventTime.IsZero() {
		eventTime = time.Now()
	}
	clientIP := entry.ClientIP
	if strings.TrimSpace(clientIP) == "" {
		clientIP = entry.RemoteIP
	}

	base := wafmodel.WafEvent{
		EventTime:     eventTime,
		TransactionID: meta.TransactionID,
		Interrupted:   meta.Interrupted,
		Status:        entry.Status,
		ClientIP:      clientIP,
		Country:       entry.Country,
		Province:      entry.Province,
		City:          entry.City,
		Host:          entry.Host,
		Method:        entry.Method,
		Uri:           entry.Uri,
		UserAgent:     entry.UserAgent,
		CaddyLogID:    entry.ID,
		RawJSON:       entry.RawLog,
	}

	if len(meta.MatchedRules) == 0 {
		event := base
		event.Severity = meta.HighestSeverity
		return []*wafmodel.WafEvent{&event}
	}

	// 同一事务同一规则可能多阶段重复命中，事务内去重只保留首条
	seen := make(map[string]struct{}, len(meta.MatchedRules))
	events := make([]*wafmodel.WafEvent, 0, len(meta.MatchedRules))
	for _, rule := range meta.MatchedRules {
		ruleID := strings.TrimSpace(shared.AsString(rule.ID))
		if _, dup := seen[ruleID]; dup {
			continue
		}
		seen[ruleID] = struct{}{}

		event := base
		event.RuleID = ruleID
		event.RuleMsg = strings.TrimSpace(rule.Msg)
		if event.RuleMsg == "" {
			event.RuleMsg = strings.TrimSpace(rule.Message)
		}
		event.Severity = shared.AsString(rule.Severity)
		if strings.TrimSpace(event.Severity) == "" {
			event.Severity = meta.HighestSeverity
		}
		events = append(events, &event)
	}
	return events
}
