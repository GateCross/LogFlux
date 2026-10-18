package caddy

import (
	"encoding/json"
	"fmt"
	"strings"

	systemmodel "logflux/model/system"

	"gorm.io/gorm"
)

const (
	logfluxManagedCaddyfileMarker = "# LogFlux managed Caddyfile"

	managedCaddyDefaultSiteAddress = ":80"
	managedCaddyDefaultBackend     = "localhost:8888"
	managedCaddyDefaultFrontend    = "/app/frontend"
	managedCaddyDefaultAccessLog   = "/var/log/caddy/access.log"
	managedCaddyDefaultWafAuditLog = "/var/log/caddy/waf_audit.log"

	// managedCRSRulesRoot 是文件系统 CRS 规则根目录，与 docker/Caddyfile 保持一致。
	//
	// coraza-caddy 在 load_owasp_crs 开启时会把 coreruleset.FS 与 OSFS 合并，
	// 其中 OSFS 直接使用 os.Open，因此这里的绝对路径可以读到本项目自己管理
	// 的规则文件，从而让 CRS 在线更新机制真正生效。
	managedCRSRulesRoot = "/config/security/current"

	// managedCRSCorazaConfigInclude 是 Coraza 基线配置的加载路径。
	managedCRSCorazaConfigInclude = managedCRSRulesRoot + "/rules/@coraza.conf-recommended"

	// managedCRSSetupConfigInclude 是 CRS setup 配置的加载路径。
	managedCRSSetupConfigInclude = managedCRSRulesRoot + "/rules/@crs-setup.conf.example"

	// managedCRSRulesInclude 是 OWASP CRS 规则文件的加载路径。
	//
	// 规则内部以 @pmFromFile php-variables.data 之类的相对文件名引用同目录的
	// 数据文件，因此规则文件与数据文件必须位于同一目录。
	managedCRSRulesInclude = managedCRSRulesRoot + "/rules/@owasp_crs/*.conf"
)

type managedCaddyfileOptions struct {
	SiteAddress        string
	Backend            string
	FrontendRoot       string
	AccessLogPath      string
	WafAuditLog        string
	WafEnabled         bool
	ForwardAuthEnabled bool
	Directives         string
}

func buildPolicyCandidateCaddyConfig(currentConfig, directives string, wafEnabled, forwardAuthEnabled bool) (string, error) {
	if shouldRenderManagedCaddyfile(currentConfig) {
		return renderManagedCaddyfile(defaultManagedCaddyfileOptions(currentConfig, directives, wafEnabled, forwardAuthEnabled))
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

func defaultManagedCaddyfileOptions(currentConfig, directives string, wafEnabled, forwardAuthEnabled bool) managedCaddyfileOptions {
	return managedCaddyfileOptions{
		SiteAddress:        selectManagedCaddySiteAddress(currentConfig),
		Backend:            managedCaddyDefaultBackend,
		FrontendRoot:       managedCaddyDefaultFrontend,
		AccessLogPath:      managedCaddyDefaultAccessLog,
		WafAuditLog:        managedCaddyDefaultWafAuditLog,
		WafEnabled:         wafEnabled,
		ForwardAuthEnabled: forwardAuthEnabled,
		Directives:         directives,
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
	if options.ForwardAuthEnabled {
		builder.WriteString(fmt.Sprintf("  forward_auth %s {\n", options.Backend))
		builder.WriteString("    uri /api/internal/geo-check\n")
		builder.WriteString("  }\n\n")
	}
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
		fmt.Sprintf("Include %s", managedCRSCorazaConfigInclude),
		fmt.Sprintf("Include %s", managedCRSSetupConfigInclude),
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
	lines = append(lines, "", fmt.Sprintf("Include %s", managedCRSRulesInclude))
	return strings.Join(lines, "\n")
}

// isManagedCorazaBaseDirective 判断是否为规则加载类基础指令。
//
// 这类指令由 LogFlux 统一生成，用户策略里重复出现时需忽略，避免把规则加载
// 路径指向别处。同时兼容历史的内嵌路径写法（@owasp_crs/*.conf），保证旧配置
// 升级后不会残留失效的 Include。
func isManagedCorazaBaseDirective(line string) bool {
	normalized := strings.ToLower(strings.TrimSpace(line))
	if !strings.HasPrefix(normalized, "include ") {
		return strings.HasPrefix(normalized, "secauditlog ")
	}
	return isManagedCRSInclude(normalized)
}

// isManagedCRSInclude 判断一条 Include 是否指向 CRS 规则加载路径。
func isManagedCRSInclude(normalizedInclude string) bool {
	return strings.HasPrefix(normalizedInclude, "include @coraza.conf") ||
		strings.HasPrefix(normalizedInclude, "include @crs-setup.conf") ||
		strings.HasPrefix(normalizedInclude, "include @owasp_crs/") ||
		strings.HasPrefix(normalizedInclude, "include "+strings.ToLower(managedCRSCorazaConfigInclude)) ||
		strings.HasPrefix(normalizedInclude, "include "+strings.ToLower(managedCRSSetupConfigInclude)) ||
		strings.HasPrefix(normalizedInclude, "include "+strings.ToLower(managedCRSRulesInclude))
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

// isIPRegionEnabled 从 DB 检查 IP 区域限制是否启用
func isIPRegionEnabled(db *gorm.DB) bool {
	model := systemmodel.NewSystemConfigModel(db)
	cfg, err := model.GetByKey("ip_region")
	if err != nil {
		return false
	}
	var ipCfg struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.Unmarshal([]byte(cfg.Value), &ipCfg); err != nil {
		return false
	}
	return ipCfg.Enabled
}
