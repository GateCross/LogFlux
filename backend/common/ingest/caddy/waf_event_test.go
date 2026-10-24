package caddy

import (
	"encoding/json"
	"testing"
	"time"

	caddymodel "logflux/model/caddy"
)

func buildWafEntry(t *testing.T, extra map[string]any) *caddymodel.CaddyLog {
	t.Helper()
	raw, err := json.Marshal(extra)
	if err != nil {
		t.Fatalf("构造 ExtraData 失败: %v", err)
	}
	return &caddymodel.CaddyLog{
		ID:        7,
		LogTime:   time.Date(2026, 10, 24, 12, 0, 0, 0, time.Local),
		ClientIP:  "1.2.3.4",
		Host:      "example.com",
		Method:    "GET",
		Uri:       "/a?b=1",
		Status:    403,
		UserAgent: "ua",
		Country:   "中国",
		Province:  "广东",
		City:      "深圳",
		RawLog:    `{"transaction":{}}`,
		ExtraData: string(raw),
	}
}

func TestWafEventsFromCaddyLog_NonWafSource(t *testing.T) {
	entry := buildWafEntry(t, map[string]any{"source": "access"})
	if events := WafEventsFromCaddyLog(entry); events != nil {
		t.Fatalf("非审计来源应返回 nil，实际 %v", events)
	}
}

func TestWafEventsFromCaddyLog_RuleHits(t *testing.T) {
	entry := buildWafEntry(t, map[string]any{
		"source":           "waf",
		"transaction_id":   "tx-1",
		"interrupted":      true,
		"highest_severity": "5",
		"matched_rules": []map[string]any{
			{"id": 942100, "msg": "SQL 注入", "severity": "3"},
			{"id": "942200", "message": "回退消息"},
			{"id": 942100, "msg": "同规则重复命中"},
		},
	})

	events := WafEventsFromCaddyLog(entry)
	if len(events) != 2 {
		t.Fatalf("去重后应产出 2 条事件，实际 %d", len(events))
	}

	first := events[0]
	if first.TransactionID != "tx-1" || first.RuleID != "942100" || first.RuleMsg != "SQL 注入" {
		t.Errorf("首条事件字段不符: %+v", first)
	}
	if !first.Interrupted || first.Status != 403 || first.CaddyLogID != 7 {
		t.Errorf("公共字段未继承: %+v", first)
	}
	if first.Country != "中国" || first.Province != "广东" {
		t.Errorf("地理信息未继承: %+v", first)
	}
	if first.EventTime != entry.LogTime {
		t.Errorf("事件时间应取日志时间")
	}

	second := events[1]
	if second.RuleID != "942200" {
		t.Errorf("字符串规则 ID 未归一化: %+v", second)
	}
	if second.RuleMsg != "回退消息" {
		t.Errorf("msg 缺失时应回退 message: %+v", second)
	}
	if second.Severity != "5" {
		t.Errorf("规则 severity 缺失时应回退事务最高等级: %+v", second)
	}
}

func TestWafEventsFromCaddyLog_NoMatchedRules(t *testing.T) {
	entry := buildWafEntry(t, map[string]any{
		"source":           "waf",
		"transaction_id":   "tx-2",
		"interrupted":      true,
		"highest_severity": "4",
	})

	events := WafEventsFromCaddyLog(entry)
	if len(events) != 1 {
		t.Fatalf("无命中规则应产出 1 条事件，实际 %d", len(events))
	}
	if events[0].RuleID != "" || events[0].Severity != "4" {
		t.Errorf("兜底事件字段不符: %+v", events[0])
	}
}

func TestWafEventsFromCaddyLog_ClientIPFallback(t *testing.T) {
	entry := buildWafEntry(t, map[string]any{"source": "waf", "transaction_id": "tx-3"})
	entry.ClientIP = ""
	entry.RemoteIP = "5.6.7.8"

	events := WafEventsFromCaddyLog(entry)
	if len(events) != 1 || events[0].ClientIP != "5.6.7.8" {
		t.Fatalf("ClientIP 缺失时应回退 RemoteIP: %+v", events)
	}
}
