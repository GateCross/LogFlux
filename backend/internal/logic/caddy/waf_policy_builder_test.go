package caddy

import (
	"context"
	"strings"
	"testing"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/model"
)

func TestBuildWafPolicyDirectivesDeterministic(t *testing.T) {
	policy := &model.WafPolicy{
		Name:                        "default-runtime",
		Enabled:                     true,
		IsDefault:                   true,
		EngineMode:                  "detectiononly",
		AuditEngine:                 "relevantonly",
		AuditLogFormat:              "json",
		AuditRelevantStatus:         "^(?:5|4(?!04))",
		RequestBodyAccess:           true,
		RequestBodyLimit:            10485760,
		RequestBodyNoFilesLimit:     1048576,
		CrsTemplate:                 "balanced",
		CrsParanoiaLevel:            2,
		CrsInboundAnomalyThreshold:  5,
		CrsOutboundAnomalyThreshold: 4,
	}

	first, err := buildWafPolicyDirectives(policy)
	if err != nil {
		t.Fatalf("buildWafPolicyDirectives first call failed: %v", err)
	}

	second, err := buildWafPolicyDirectives(policy)
	if err != nil {
		t.Fatalf("buildWafPolicyDirectives second call failed: %v", err)
	}

	if first != second {
		t.Fatalf("expected deterministic output, first=%q second=%q", first, second)
	}

	expectedFragments := []string{
		"SecRuleEngine DetectionOnly",
		"SecAuditEngine RelevantOnly",
		"SecAuditLogFormat JSON",
		"SecAuditLogRelevantStatus ^(?:5|4(?!04))",
		"SecRequestBodyAccess On",
		"SecRequestBodyLimit 10485760",
		"SecRequestBodyNoFilesLimit 1048576",
		`SecAction "id:900000,phase:1,pass,nolog,t:none,setvar:tx.paranoia_level=2"`,
		`SecAction "id:900110,phase:1,pass,nolog,t:none,setvar:tx.inbound_anomaly_score_threshold=5"`,
		`SecAction "id:900100,phase:1,pass,nolog,t:none,setvar:tx.outbound_anomaly_score_threshold=4"`,
	}
	for _, fragment := range expectedFragments {
		if !strings.Contains(first, fragment) {
			t.Fatalf("directives missing fragment %q, output=%q", fragment, first)
		}
	}
}

func TestBuildWafPolicyDirectivesInvalidEnum(t *testing.T) {
	policy := &model.WafPolicy{
		EngineMode:                  "invalid_mode",
		AuditEngine:                 "relevantonly",
		AuditLogFormat:              "json",
		AuditRelevantStatus:         "^(?:5|4(?!04))",
		RequestBodyAccess:           true,
		RequestBodyLimit:            10485760,
		RequestBodyNoFilesLimit:     1048576,
		CrsTemplate:                 "low_fp",
		CrsParanoiaLevel:            1,
		CrsInboundAnomalyThreshold:  10,
		CrsOutboundAnomalyThreshold: 8,
	}

	_, err := buildWafPolicyDirectives(policy)
	if err == nil {
		t.Fatalf("expected invalid engine mode error")
	}
	if !strings.Contains(err.Error(), "invalid engine mode") {
		t.Fatalf("expected invalid engine mode error, got %v", err)
	}
}

func TestBuildWafPolicyDirectivesOutOfRange(t *testing.T) {
	policy := &model.WafPolicy{
		EngineMode:                  "on",
		AuditEngine:                 "relevantonly",
		AuditLogFormat:              "json",
		AuditRelevantStatus:         "^(?:5|4(?!04))",
		RequestBodyAccess:           true,
		RequestBodyLimit:            1024*1024*1024 + 1,
		RequestBodyNoFilesLimit:     1048576,
		CrsTemplate:                 "low_fp",
		CrsParanoiaLevel:            1,
		CrsInboundAnomalyThreshold:  10,
		CrsOutboundAnomalyThreshold: 8,
	}

	_, err := buildWafPolicyDirectives(policy)
	if err == nil {
		t.Fatalf("expected requestBodyLimit out-of-range error")
	}
	if !strings.Contains(err.Error(), "requestBodyLimit is too large") {
		t.Fatalf("expected requestBodyLimit too large error, got %v", err)
	}
}

func TestApplyPolicyReqToModelOutOfRange(t *testing.T) {
	helper := newWafLogicHelper(context.Background(), &svc.ServiceContext{}, nil)
	req := &types.WafPolicyReq{
		Name:                        "test-policy",
		EngineMode:                  "on",
		AuditEngine:                 "relevantonly",
		AuditLogFormat:              "json",
		AuditRelevantStatus:         "^(?:5|4(?!04))",
		RequestBodyAccess:           true,
		RequestBodyLimit:            1024*1024*1024 + 1,
		RequestBodyNoFilesLimit:     1048576,
		CrsTemplate:                 "balanced",
		CrsParanoiaLevel:            2,
		CrsInboundAnomalyThreshold:  5,
		CrsOutboundAnomalyThreshold: 4,
	}
	policy := &model.WafPolicy{}

	err := applyPolicyReqToModel(helper, req, policy)
	if err == nil {
		t.Fatalf("expected applyPolicyReqToModel out-of-range error")
	}
	if !strings.Contains(err.Error(), "requestBodyLimit is too large") {
		t.Fatalf("expected requestBodyLimit too large error, got %v", err)
	}
}

func TestApplyPolicyReqToModelCRSParanoiaOutOfRange(t *testing.T) {
	helper := newWafLogicHelper(context.Background(), &svc.ServiceContext{}, nil)
	req := &types.WafPolicyReq{
		Name:                        "test-policy",
		EngineMode:                  "on",
		AuditEngine:                 "relevantonly",
		AuditLogFormat:              "json",
		AuditRelevantStatus:         "^(?:5|4(?!04))",
		RequestBodyAccess:           true,
		RequestBodyLimit:            10 * 1024 * 1024,
		RequestBodyNoFilesLimit:     1024 * 1024,
		CrsTemplate:                 "balanced",
		CrsParanoiaLevel:            8,
		CrsInboundAnomalyThreshold:  5,
		CrsOutboundAnomalyThreshold: 4,
	}
	policy := &model.WafPolicy{}

	err := applyPolicyReqToModel(helper, req, policy)
	if err == nil {
		t.Fatalf("expected applyPolicyReqToModel crsParanoiaLevel out-of-range error")
	}
	if !strings.Contains(err.Error(), "crsParanoiaLevel must be between 1 and 4") {
		t.Fatalf("expected crsParanoiaLevel range error, got %v", err)
	}
}
