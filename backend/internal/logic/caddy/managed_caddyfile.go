package caddy

import (
	"fmt"
	"strings"
)

const (
	logfluxManagedCaddyfileMarker = "# LogFlux managed Caddyfile"

	managedCaddyDefaultSiteAddress = ":80"
	managedCaddyDefaultBackend     = "localhost:8888"
	managedCaddyDefaultFrontend    = "/app/frontend"
	managedCaddyDefaultAccessLog   = "/var/log/caddy/access.log"
	managedCaddyDefaultWafAuditLog = "/var/log/caddy/waf_audit.log"
)

type managedCaddyfileOptions struct {
	SiteAddress   string
	Backend       string
	FrontendRoot  string
	AccessLogPath string
	WafAuditLog   string
	WafEnabled    bool
	Directives    string
}

func buildPolicyCandidateCaddyConfig(currentConfig, directives string, wafEnabled bool) (string, error) {
	if shouldRenderManagedCaddyfile(currentConfig) {
		return renderManagedCaddyfile(defaultManagedCaddyfileOptions(currentConfig, directives, wafEnabled))
	}
	return applyWafPolicyToCaddyConfig(currentConfig, directives)
}

func shouldRenderManagedCaddyfile(config string) bool {
	trimmed := strings.TrimSpace(config)
	if trimmed == "" {
		return true
	}
	if strings.Contains(trimmed, logfluxManagedCaddyfileMarker) {
		return true
	}
	if strings.Contains(trimmed, "(frontend_full)") || strings.Contains(trimmed, "(frontend_simple)") {
		return true
	}
	return strings.Contains(trimmed, "root * /app/frontend") && strings.Contains(trimmed, "reverse_proxy localhost:8888")
}

func defaultManagedCaddyfileOptions(currentConfig, directives string, wafEnabled bool) managedCaddyfileOptions {
	return managedCaddyfileOptions{
		SiteAddress:   selectManagedCaddySiteAddress(currentConfig),
		Backend:       managedCaddyDefaultBackend,
		FrontendRoot:  managedCaddyDefaultFrontend,
		AccessLogPath: managedCaddyDefaultAccessLog,
		WafAuditLog:   managedCaddyDefaultWafAuditLog,
		WafEnabled:    wafEnabled,
		Directives:    directives,
	}
}

func selectManagedCaddySiteAddress(config string) string {
	blocks, _, err := parseTopLevelCaddyBlocks(config)
	if err != nil {
		return managedCaddyDefaultSiteAddress
	}

	firstSite := ""
	for _, block := range blocks {
		if block.Kind != "site" {
			continue
		}
		address := strings.TrimSpace(block.Address)
		if address == "" {
			continue
		}
		if address == managedCaddyDefaultSiteAddress {
			return managedCaddyDefaultSiteAddress
		}
		if firstSite == "" {
			firstSite = address
		}
	}
	if firstSite != "" {
		return firstSite
	}
	return managedCaddyDefaultSiteAddress
}

func renderManagedCaddyfile(options managedCaddyfileOptions) (string, error) {
	options = normalizeManagedCaddyfileOptions(options)
	if options.WafEnabled && strings.TrimSpace(options.Directives) == "" {
		return "", fmt.Errorf("WAF 策略指令为空")
	}

	var builder strings.Builder
	builder.WriteString(logfluxManagedCaddyfileMarker + "\n")
	builder.WriteString("{\n")
	builder.WriteString("  admin :2019\n")
	if options.WafEnabled {
		builder.WriteString("  order coraza_waf first\n")
	}
	builder.WriteString("}\n\n")

	if options.WafEnabled {
		builder.WriteString(renderManagedWafSnippet(options.Directives, options.WafAuditLog))
		builder.WriteString("\n")
	}

	builder.WriteString(options.SiteAddress + " {\n")
	if options.WafEnabled {
		builder.WriteString("  import waf_protect\n\n")
	}
	builder.WriteString("  log {\n")
	builder.WriteString(fmt.Sprintf("    output file %s\n", options.AccessLogPath))
	builder.WriteString("    format json\n")
	builder.WriteString("  }\n\n")
	builder.WriteString("  encode gzip\n\n")
	builder.WriteString("  handle /api/health {\n")
	builder.WriteString("    respond \"OK\" 200\n")
	builder.WriteString("  }\n\n")
	builder.WriteString("  handle /api/* {\n")
	builder.WriteString(fmt.Sprintf("    reverse_proxy %s\n", options.Backend))
	builder.WriteString("  }\n\n")
	builder.WriteString("  handle {\n")
	builder.WriteString(fmt.Sprintf("    root * %s\n", options.FrontendRoot))
	builder.WriteString("    try_files {path} /index.html\n")
	builder.WriteString("    file_server\n")
	builder.WriteString("  }\n")
	builder.WriteString("}\n")

	return builder.String(), nil
}

func normalizeManagedCaddyfileOptions(options managedCaddyfileOptions) managedCaddyfileOptions {
	options.SiteAddress = strings.TrimSpace(options.SiteAddress)
	if options.SiteAddress == "" {
		options.SiteAddress = managedCaddyDefaultSiteAddress
	}
	options.Backend = strings.TrimSpace(options.Backend)
	if options.Backend == "" {
		options.Backend = managedCaddyDefaultBackend
	}
	options.FrontendRoot = strings.TrimSpace(options.FrontendRoot)
	if options.FrontendRoot == "" {
		options.FrontendRoot = managedCaddyDefaultFrontend
	}
	options.AccessLogPath = strings.TrimSpace(options.AccessLogPath)
	if options.AccessLogPath == "" {
		options.AccessLogPath = managedCaddyDefaultAccessLog
	}
	options.WafAuditLog = strings.TrimSpace(options.WafAuditLog)
	if options.WafAuditLog == "" {
		options.WafAuditLog = managedCaddyDefaultWafAuditLog
	}
	options.Directives = strings.TrimSpace(options.Directives)
	return options
}

func renderManagedWafSnippet(directives, auditLogPath string) string {
	return strings.Join([]string{
		"(waf_protect) {",
		"  coraza_waf {",
		"    load_owasp_crs",
		"    directives `",
		indentManagedCorazaDirectives(composeManagedCorazaDirectives(directives, auditLogPath)),
		"    `",
		"  }",
		"}",
	}, "\n") + "\n"
}

func composeManagedCorazaDirectives(directives, auditLogPath string) string {
	lines := []string{
		"Include @coraza.conf-recommended",
		"Include @crs-setup.conf.example",
		"",
	}

	for _, line := range strings.Split(strings.TrimSpace(directives), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if isManagedCorazaBaseDirective(trimmed) {
			continue
		}
		lines = append(lines, trimmed)
	}

	if !containsDirectiveName(lines, "SecAuditLog") {
		lines = append(lines, fmt.Sprintf("SecAuditLog %s", strings.TrimSpace(auditLogPath)))
	}
	lines = append(lines, "", "Include @owasp_crs/*.conf")
	return strings.Join(lines, "\n")
}

func isManagedCorazaBaseDirective(line string) bool {
	normalized := strings.ToLower(strings.TrimSpace(line))
	return strings.HasPrefix(normalized, "include @coraza.conf") ||
		strings.HasPrefix(normalized, "include @crs-setup.conf") ||
		strings.HasPrefix(normalized, "include @owasp_crs/") ||
		strings.HasPrefix(normalized, "secauditlog ")
}

func containsDirectiveName(lines []string, directiveName string) bool {
	expected := strings.ToLower(strings.TrimSpace(directiveName))
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		if strings.ToLower(fields[0]) == expected {
			return true
		}
	}
	return false
}

func indentManagedCorazaDirectives(directives string) string {
	lines := strings.Split(strings.TrimSpace(directives), "\n")
	for idx, line := range lines {
		if strings.TrimSpace(line) == "" {
			lines[idx] = ""
			continue
		}
		lines[idx] = "      " + strings.TrimSpace(line)
	}
	return strings.Join(lines, "\n")
}
