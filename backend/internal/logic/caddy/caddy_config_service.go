package caddy

import (
	"encoding/json"
	"errors"
	"fmt"
	caddymodel "logflux/model/caddy"
	"regexp"
	"strconv"
	"strings"
)

const (
	caddyConfigModeQuick = "quick"
	caddyConfigModeRaw   = "raw"
)

type caddyConfigService struct{}

type caddyConfigPrepareInput struct {
	Mode    string
	Config  string
	Modules string
}

type caddyConfigPrepareResult struct {
	Config  string
	Modules string
	Actions []string
}

type caddyFormModel struct {
	SchemaVersion int             `json:"schemaVersion,omitempty"`
	Global        caddyGlobal     `json:"global"`
	Upstreams     []caddyUpstream `json:"upstreams"`
	Sites         []caddySite     `json:"sites"`
}

type caddyGlobal struct {
	Raw string `json:"raw,omitempty"`
}

type caddyUpstream struct {
	Name        string            `json:"name"`
	Targets     []string          `json:"targets"`
	LBPolicy    string            `json:"lbPolicy,omitempty"`
	HealthCheck *caddyHealthCheck `json:"healthCheck,omitempty"`
}

type caddySite struct {
	ID         string       `json:"id"`
	Name       string       `json:"name"`
	Enabled    bool         `json:"enabled"`
	Domains    []string     `json:"domains"`
	TLS        *caddyTLS    `json:"tls,omitempty"`
	Imports    []string     `json:"imports,omitempty"`
	GeoIP2Vars []string     `json:"geoip2Vars,omitempty"`
	Encode     []string     `json:"encode,omitempty"`
	Routes     []caddyRoute `json:"routes"`
}

type caddyRoute struct {
	ID        string        `json:"id"`
	Name      string        `json:"name"`
	Enabled   bool          `json:"enabled"`
	Match     caddyMatch    `json:"match"`
	Handles   []caddyHandle `json:"handles"`
	LogAppend []caddyKV     `json:"logAppend,omitempty"`
}

type caddyMatch struct {
	Host       []string  `json:"host"`
	Path       []string  `json:"path"`
	Method     []string  `json:"method"`
	Header     []caddyKV `json:"header"`
	Query      []caddyKV `json:"query"`
	Expression string    `json:"expression,omitempty"`
}

type caddyHandle struct {
	ID                    string            `json:"id"`
	Type                  string            `json:"type"`
	Enabled               bool              `json:"enabled"`
	Upstream              string            `json:"upstream,omitempty"`
	LBPolicy              string            `json:"lbPolicy,omitempty"`
	HealthCheck           *caddyHealthCheck `json:"healthCheck,omitempty"`
	TransportProtocol     string            `json:"transportProtocol,omitempty"`
	TLSInsecureSkipVerify bool              `json:"tlsInsecureSkipVerify,omitempty"`
	Root                  string            `json:"root,omitempty"`
	Browse                bool              `json:"browse,omitempty"`
	Status                int               `json:"status,omitempty"`
	Body                  string            `json:"body,omitempty"`
	To                    string            `json:"to,omitempty"`
	Code                  int               `json:"code,omitempty"`
	Rules                 []caddyHeaderRule `json:"rules,omitempty"`
	URI                   string            `json:"uri,omitempty"`
}

type caddyTLS struct {
	Mode     string `json:"mode,omitempty"`
	CertFile string `json:"certFile,omitempty"`
	KeyFile  string `json:"keyFile,omitempty"`
}

type caddyHealthCheck struct {
	Path     string `json:"path"`
	Interval string `json:"interval,omitempty"`
	Timeout  string `json:"timeout,omitempty"`
}

type caddyHeaderRule struct {
	Op    string `json:"op"`
	Key   string `json:"key"`
	Value string `json:"value,omitempty"`
}

type caddyKV struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

var (
	caddyDomainPattern   = regexp.MustCompile(`^(\*\.)?([a-zA-Z0-9-]+\.)+[a-zA-Z0-9-]+$`)
	caddyPortOnlyPattern = regexp.MustCompile(`^:\d+$`)
	caddyMethodAllowSet  = map[string]struct{}{
		"GET": {}, "POST": {}, "PUT": {}, "PATCH": {}, "DELETE": {}, "HEAD": {}, "OPTIONS": {},
	}
)

func newCaddyConfigService() *caddyConfigService {
	return &caddyConfigService{}
}

func (s *caddyConfigService) Prepare(input caddyConfigPrepareInput) (*caddyConfigPrepareResult, error) {
	mode := normalizeCaddyConfigMode(input.Mode, input.Modules)
	switch mode {
	case caddyConfigModeQuick:
		return s.prepareQuick(input.Config, input.Modules)
	case caddyConfigModeRaw:
		return s.prepareRaw(input.Config, input.Modules)
	default:
		return nil, fmt.Errorf("Caddy 配置模式不支持: %s", mode)
	}
}

func (s *caddyConfigService) prepareQuick(config, modules string) (*caddyConfigPrepareResult, error) {
	model, normalizedModules, err := decodeCaddyModules(modules)
	if err != nil {
		return nil, err
	}
	if errs := validateCaddyFormModel(model); len(errs) > 0 {
		return nil, errors.New(strings.Join(errs, "；"))
	}

	// quick 模式下，前端会把结构化站点和只读保留块合并成完整 Caddyfile。
	// 后端只用 modules 重新渲染会丢失 snippet/复杂保留块，因此优先使用请求中的完整 config。
	caddyfile := config
	if strings.TrimSpace(caddyfile) == "" {
		caddyfile = renderCaddyFormModel(model)
	}
	if strings.TrimSpace(caddyfile) == "" {
		return nil, fmt.Errorf("结构化配置生成结果为空")
	}
	caddyfile, wafActions, err := ensureQuickWafImportDependencies(caddyfile)
	if err != nil {
		return nil, err
	}
	if updated, changed := ensureApiCrsExclusion(caddyfile); changed {
		caddyfile = updated
		wafActions = append(wafActions, "自动注入 /api/ CRS 排除规则，防止 WAF 误拦 LogFlux 配置保存")
	}
	if updated, changed := ensureCrsFalsePositiveExclusions(caddyfile); changed {
		caddyfile = updated
		wafActions = append(wafActions, "自动注入 CRS 误报排除规则（IP 访问 / Cookie 误报）")
	}
	actions := []string{
		"根据结构化配置生成 Caddyfile",
		"复杂自定义片段保留在全局配置或站点 import 中",
	}
	actions = append(actions, wafActions...)
	return &caddyConfigPrepareResult{
		Config:  formatCaddyfile(caddyfile),
		Modules: normalizedModules,
		Actions: actions,
	}, nil
}

func ensureQuickWafImportDependencies(config string) (string, []string, error) {
	snapshot, err := inspectWafIntegration(config)
	if err != nil {
		return "", nil, err
	}
	if len(snapshot.ImportedSites) == 0 {
		return config, nil, nil
	}

	nextConfig := config
	actions := make([]string, 0, 2)
	if !snapshot.OrderReady {
		var changed bool
		nextConfig, changed, err = ensureCorazaOrder(nextConfig)
		if err != nil {
			return "", nil, err
		}
		if changed {
			actions = append(actions, "补齐全局 order coraza_waf first")
		}
	}
	if !snapshot.SnippetReady || !snapshot.DirectiveReady {
		var changed bool
		nextConfig, changed, err = ensureWafProtectSnippet(nextConfig)
		if err != nil {
			return "", nil, err
		}
		if changed {
			actions = append(actions, "补齐 waf_protect 统一片段")
		}
	}
	var changed bool
	nextConfig, changed, err = ensureWafProtectSnippetBeforeSites(nextConfig)
	if err != nil {
		return "", nil, err
	}
	if changed {
		actions = append(actions, "调整 waf_protect 片段到站点之前")
	}
	return nextConfig, actions, nil
}

func (s *caddyConfigService) prepareRaw(config, modules string) (*caddyConfigPrepareResult, error) {
	formatted := formatCaddyfile(config)
	if strings.TrimSpace(formatted) == "" {
		return nil, fmt.Errorf("Caddy 配置不能为空")
	}
	actions := []string{"使用原始 Caddyfile 内容"}
	if updated, changed := ensureApiCrsExclusion(formatted); changed {
		formatted = updated
		actions = append(actions, "自动注入 /api/ CRS 排除规则，防止 WAF 误拦 LogFlux 配置保存")
	}
	if updated, changed := ensureCrsFalsePositiveExclusions(formatted); changed {
		formatted = updated
		actions = append(actions, "自动注入 CRS 误报排除规则（IP 访问 / Cookie 误报）")
	}
	return &caddyConfigPrepareResult{
		Config:  formatted,
		Modules: normalizeCaddyModulesJSON(modules),
		Actions: actions,
	}, nil
}

func normalizeCaddyConfigMode(mode, modules string) string {
	mode = strings.ToLower(strings.TrimSpace(mode))
	if mode == "" {
		if strings.TrimSpace(modules) != "" {
			return caddyConfigModeQuick
		}
		return caddyConfigModeRaw
	}
	return mode
}

func decodeCaddyModules(modules string) (*caddyFormModel, string, error) {
	trimmed := strings.TrimSpace(modules)
	if trimmed == "" {
		return nil, emptyModulesJSON, fmt.Errorf("结构化配置不能为空")
	}
	var model caddyFormModel
	if err := json.Unmarshal([]byte(trimmed), &model); err != nil {
		return nil, emptyModulesJSON, fmt.Errorf("结构化配置 JSON 无效: %w", err)
	}
	if model.SchemaVersion == 0 {
		model.SchemaVersion = 1
	}
	if model.Upstreams == nil {
		model.Upstreams = []caddyUpstream{}
	}
	if model.Sites == nil {
		model.Sites = []caddySite{}
	}
	normalized, err := json.Marshal(model)
	if err != nil {
		return nil, emptyModulesJSON, fmt.Errorf("结构化配置标准化失败: %w", err)
	}
	return &model, string(normalized), nil
}

func validateCaddyFormModel(model *caddyFormModel) []string {
	if model == nil {
		return []string{"结构化配置不能为空"}
	}
	errors := make([]string, 0)
	enabledSites := 0
	if strings.TrimSpace(model.Global.Raw) == "" {
		for _, site := range model.Sites {
			if site.Enabled {
				enabledSites++
			}
		}
		if enabledSites == 0 {
			errors = append(errors, "至少需要一个启用站点或全局配置")
		}
	}
	upstreamNames := make(map[string]struct{}, len(model.Upstreams))
	for _, upstream := range model.Upstreams {
		name := strings.TrimSpace(upstream.Name)
		if name == "" {
			errors = append(errors, "上游名称不能为空")
			continue
		}
		if _, ok := upstreamNames[name]; ok {
			errors = append(errors, fmt.Sprintf("上游名称重复: %s", name))
		}
		upstreamNames[name] = struct{}{}
		if len(trimStringSlice(upstream.Targets)) == 0 {
			errors = append(errors, fmt.Sprintf("上游 %s 至少配置一个目标", name))
		}
	}
	for _, site := range model.Sites {
		if !site.Enabled {
			continue
		}
		siteName := firstNonEmpty(site.Name, site.ID, "未命名站点")
		domains := trimStringSlice(site.Domains)
		if len(domains) == 0 {
			errors = append(errors, fmt.Sprintf("站点 %s 至少配置一个域名", siteName))
		}
		for _, domain := range domains {
			if !isValidCaddySiteAddress(domain) {
				errors = append(errors, fmt.Sprintf("站点 %s 域名格式不合法: %s", siteName, domain))
			}
		}
		if site.TLS != nil && site.TLS.Mode == "manual" && (strings.TrimSpace(site.TLS.CertFile) == "" || strings.TrimSpace(site.TLS.KeyFile) == "") {
			errors = append(errors, fmt.Sprintf("站点 %s TLS 手动模式需填写证书和私钥", siteName))
		}
		hasRoute := false
		for _, route := range site.Routes {
			if !route.Enabled {
				continue
			}
			hasRoute = true
			errors = append(errors, validateCaddyRoute(siteName, route, upstreamNames)...)
		}
		if !hasRoute && len(trimStringSlice(site.Imports)) == 0 {
			errors = append(errors, fmt.Sprintf("站点 %s 至少配置一个路由或 import", siteName))
		}
	}
	return errors
}

func validateCaddyRoute(siteName string, route caddyRoute, upstreamNames map[string]struct{}) []string {
	errors := make([]string, 0)
	routeName := firstNonEmpty(route.Name, route.ID, "未命名路由")
	if strings.TrimSpace(route.Name) == "" {
		errors = append(errors, fmt.Sprintf("站点 %s 有未命名路由", siteName))
	}
	if len(route.Handles) == 0 {
		errors = append(errors, fmt.Sprintf("路由 %s 至少一个 Handler", routeName))
	}
	enabledHandles := 0
	for _, path := range trimStringSlice(route.Match.Path) {
		if !isValidCaddyPathPattern(path) {
			errors = append(errors, fmt.Sprintf("路由 %s Path 格式不合法: %s", routeName, path))
		}
	}
	for _, method := range trimStringSlice(route.Match.Method) {
		if _, ok := caddyMethodAllowSet[strings.ToUpper(method)]; !ok {
			errors = append(errors, fmt.Sprintf("路由 %s Method 非法: %s", routeName, method))
		}
	}
	for _, handle := range route.Handles {
		if !handle.Enabled {
			continue
		}
		enabledHandles++
		if handle.Type == "reverse_proxy" && strings.TrimSpace(handle.Upstream) == "" {
			errors = append(errors, fmt.Sprintf("路由 %s 的 reverse_proxy 未选择上游", routeName))
		}
		if handle.Type == "reverse_proxy" {
			if _, ok := upstreamNames[strings.TrimSpace(handle.Upstream)]; ok {
				continue
			}
		}
	}
	if len(route.Handles) > 0 && enabledHandles == 0 {
		errors = append(errors, fmt.Sprintf("路由 %s 至少启用一个 Handler", routeName))
	}
	return errors
}

func renderCaddyFormModel(model *caddyFormModel) string {
	lines := make([]string, 0)
	globalRaw := strings.TrimSpace(model.Global.Raw)
	if globalRaw != "" {
		lines = append(lines, globalRaw, "")
	}
	if len(model.Sites) == 0 {
		if len(lines) > 0 {
			return strings.TrimSpace(strings.Join(lines, "\n"))
		}
		return "# No sites defined"
	}
	upstreams := make(map[string]caddyUpstream, len(model.Upstreams))
	for _, upstream := range model.Upstreams {
		upstreams[strings.TrimSpace(upstream.Name)] = upstream
	}
	for _, site := range model.Sites {
		if !site.Enabled {
			continue
		}
		renderCaddySite(&lines, site, upstreams)
	}
	result := strings.TrimSpace(strings.Join(lines, "\n"))
	if result == "" {
		return "# No routes defined"
	}
	return result
}

func renderCaddySite(lines *[]string, site caddySite, upstreams map[string]caddyUpstream) {
	domains := trimStringSlice(site.Domains)
	if len(domains) == 0 {
		return
	}
	*lines = append(*lines, strings.Join(domains, " ")+" {")
	for _, item := range trimStringSlice(site.GeoIP2Vars) {
		*lines = append(*lines, "  geoip2_vars "+item)
	}
	for _, item := range trimStringSlice(site.Imports) {
		*lines = append(*lines, "  import "+item)
	}
	if encodes := trimStringSlice(site.Encode); len(encodes) > 0 {
		*lines = append(*lines, "  encode "+strings.Join(encodes, " "))
	}
	renderCaddyTLS(lines, site.TLS)

	usedMatchers := make(map[string]struct{})
	for _, route := range site.Routes {
		if route.Enabled {
			renderCaddyRoute(lines, route, upstreams, usedMatchers)
		}
	}
	*lines = append(*lines, "}", "")
}

func renderCaddyTLS(lines *[]string, tls *caddyTLS) {
	if tls == nil || strings.TrimSpace(tls.Mode) == "" || tls.Mode == "auto" {
		return
	}
	switch tls.Mode {
	case "off":
		*lines = append(*lines, "  tls off")
	case "internal":
		*lines = append(*lines, "  tls internal")
	case "manual":
		if strings.TrimSpace(tls.CertFile) != "" && strings.TrimSpace(tls.KeyFile) != "" {
			*lines = append(*lines, fmt.Sprintf("  tls %s %s", strings.TrimSpace(tls.CertFile), strings.TrimSpace(tls.KeyFile)))
		}
	}
}

func renderCaddyRoute(lines *[]string, route caddyRoute, upstreams map[string]caddyUpstream, usedMatchers map[string]struct{}) {
	matcherLines := buildCaddyMatcherLines(route.Match)
	matcherName := selectCaddyMatcherName(route, usedMatchers)
	if len(matcherLines) > 0 {
		usedMatchers[matcherName] = struct{}{}
	}
	enabledHandles := enabledCaddyHandles(route.Handles)
	headerOnly := len(enabledHandles) > 0 && handlesAllType(enabledHandles, "header") && len(route.LogAppend) == 0
	fileServerOnly := len(enabledHandles) > 0 && handlesAllType(enabledHandles, "file_server") && len(route.LogAppend) == 0

	switch {
	case len(matcherLines) > 0:
		*lines = append(*lines, "  "+matcherName+" {")
		for _, line := range matcherLines {
			*lines = append(*lines, "    "+line)
		}
		*lines = append(*lines, "  }")
		if headerOnly {
			renderCaddyHeaderOnly(lines, "  header "+matcherName+" ", enabledHandles)
			return
		}
		*lines = append(*lines, "  handle "+matcherName+" {")
	case headerOnly:
		renderCaddyHeaderOnly(lines, "  header ", enabledHandles)
		return
	case fileServerOnly:
		for _, handle := range enabledHandles {
			if strings.TrimSpace(handle.Root) != "" {
				*lines = append(*lines, "  root * "+strings.TrimSpace(handle.Root))
			}
			*lines = append(*lines, "  file_server"+browseSuffix(handle.Browse))
		}
		return
	default:
		*lines = append(*lines, "  handle {")
	}

	for _, handle := range enabledHandles {
		renderCaddyHandle(lines, handle, upstreams)
	}
	for _, item := range route.LogAppend {
		if strings.TrimSpace(item.Key) != "" {
			*lines = append(*lines, strings.TrimRight("    log_append "+strings.TrimSpace(item.Key)+" "+strings.TrimSpace(item.Value), " "))
		}
	}
	*lines = append(*lines, "  }")
}

func buildCaddyMatcherLines(match caddyMatch) []string {
	lines := make([]string, 0)
	if values := trimStringSlice(match.Host); len(values) > 0 {
		lines = append(lines, "host "+strings.Join(values, " "))
	}
	if values := trimStringSlice(match.Path); len(values) > 0 {
		lines = append(lines, "path "+strings.Join(values, " "))
	}
	if values := trimStringSlice(match.Method); len(values) > 0 {
		lines = append(lines, "method "+strings.Join(values, " "))
	}
	if len(match.Header) > 0 {
		parts := make([]string, 0, len(match.Header))
		for _, item := range match.Header {
			if strings.TrimSpace(item.Key) != "" {
				parts = append(parts, strings.TrimSpace(item.Key)+" "+strings.TrimSpace(item.Value))
			}
		}
		if len(parts) > 0 {
			lines = append(lines, "header "+strings.Join(parts, " "))
		}
	}
	if len(match.Query) > 0 {
		parts := make([]string, 0, len(match.Query))
		for _, item := range match.Query {
			if strings.TrimSpace(item.Key) != "" {
				parts = append(parts, strings.TrimSpace(item.Key)+"="+strings.TrimSpace(item.Value))
			}
		}
		if len(parts) > 0 {
			lines = append(lines, "query "+strings.Join(parts, " "))
		}
	}
	if strings.TrimSpace(match.Expression) != "" {
		lines = append(lines, "expression "+strings.TrimSpace(match.Expression))
	}
	return lines
}

func selectCaddyMatcherName(route caddyRoute, used map[string]struct{}) string {
	name := strings.TrimSpace(route.Name)
	if strings.HasPrefix(name, "@") && regexp.MustCompile(`^@[a-zA-Z0-9_-]+$`).MatchString(name) {
		return name
	}
	baseID := strings.TrimSpace(route.ID)
	if len(baseID) > 6 {
		baseID = baseID[:6]
	}
	if baseID == "" {
		baseID = "route"
	}
	base := "@m_" + baseID
	name = base
	for idx := 1; ; idx++ {
		if _, ok := used[name]; !ok {
			return name
		}
		name = fmt.Sprintf("%s_%d", base, idx)
	}
}

func renderCaddyHandle(lines *[]string, handle caddyHandle, upstreams map[string]caddyUpstream) {
	switch handle.Type {
	case "reverse_proxy":
		renderCaddyReverseProxy(lines, handle, upstreams)
	case "file_server":
		if strings.TrimSpace(handle.Root) != "" {
			*lines = append(*lines, "    root * "+strings.TrimSpace(handle.Root))
		}
		*lines = append(*lines, "    file_server"+browseSuffix(handle.Browse))
	case "respond":
		status := handle.Status
		if status == 0 {
			status = 200
		}
		*lines = append(*lines, fmt.Sprintf("    respond %q %d", handle.Body, status))
	case "redirect":
		code := handle.Code
		if code == 0 {
			code = 302
		}
		*lines = append(*lines, fmt.Sprintf("    redir %s %s", firstNonEmpty(handle.To, "/"), caddyRedirectCode(code)))
	case "header":
		for _, rule := range handle.Rules {
			*lines = append(*lines, "    "+renderHeaderRule("header ", rule))
		}
	case "rewrite":
		*lines = append(*lines, "    rewrite * "+firstNonEmpty(handle.URI, "/"))
	}
}

func renderCaddyReverseProxy(lines *[]string, handle caddyHandle, upstreams map[string]caddyUpstream) {
	targets := strings.TrimSpace(handle.Upstream)
	if upstream, ok := upstreams[targets]; ok && len(trimStringSlice(upstream.Targets)) > 0 {
		targets = strings.Join(trimStringSlice(upstream.Targets), " ")
	}
	if targets == "" {
		targets = "localhost:8080"
	}
	transport := strings.TrimSpace(handle.TransportProtocol)
	if transport == "" && handle.TLSInsecureSkipVerify {
		transport = "http"
	}
	if transport != "" || handle.TLSInsecureSkipVerify {
		*lines = append(*lines, "    reverse_proxy "+targets+" {")
		if transport != "" {
			*lines = append(*lines, "      transport "+transport+" {")
			if handle.TLSInsecureSkipVerify {
				*lines = append(*lines, "        tls_insecure_skip_verify")
			}
			*lines = append(*lines, "      }")
		}
		*lines = append(*lines, "    }")
		return
	}
	*lines = append(*lines, "    reverse_proxy "+targets)
}

func renderCaddyHeaderOnly(lines *[]string, prefix string, handles []caddyHandle) {
	for _, handle := range handles {
		for _, rule := range handle.Rules {
			*lines = append(*lines, strings.TrimRight(prefix+strings.TrimPrefix(renderHeaderRule("", rule), " "), " "))
		}
	}
}

func renderHeaderRule(prefix string, rule caddyHeaderRule) string {
	key := strings.TrimSpace(rule.Key)
	value := strings.TrimSpace(rule.Value)
	if strings.EqualFold(rule.Op, "delete") {
		return prefix + "-" + key
	}
	return strings.TrimRight(prefix+key+" "+value, " ")
}

func formatCaddyfile(content string) string {
	if strings.TrimSpace(content) == "" {
		return content
	}
	lines := strings.Split(content, "\n")
	out := make([]string, 0, len(lines))
	indent := 0
	for _, raw := range lines {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			out = append(out, "")
			continue
		}
		openCount, closeCount := countCaddyLineBraces(raw)
		nextIndent := indent - closeCount
		if nextIndent < 0 {
			nextIndent = 0
		}
		out = append(out, strings.Repeat("  ", nextIndent)+trimmed)
		indent = nextIndent + openCount
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}

func countCaddyLineBraces(line string) (int, int) {
	sanitized := stripCaddyComment(line)
	openCount := 0
	closeCount := 0
	inQuote := false
	escaped := false
	for idx := 0; idx < len(sanitized); idx++ {
		ch := sanitized[idx]
		if !escaped && ch == '"' {
			inQuote = !inQuote
		}
		if !inQuote && (ch == '{' || ch == '}') && isBraceBoundary(sanitized, idx) {
			if ch == '{' {
				openCount++
			} else {
				closeCount++
			}
		}
		escaped = ch == '\\' && !escaped
		if ch != '\\' {
			escaped = false
		}
	}
	return openCount, closeCount
}

func stripCaddyComment(line string) string {
	inQuote := false
	escaped := false
	var builder strings.Builder
	for _, ch := range line {
		if !escaped && ch == '"' {
			inQuote = !inQuote
		}
		if !inQuote && ch == '#' {
			break
		}
		builder.WriteRune(ch)
		escaped = ch == '\\' && !escaped
		if ch != '\\' {
			escaped = false
		}
	}
	return builder.String()
}

func isBraceBoundary(line string, idx int) bool {
	prevOK := idx == 0 || isCaddyWhitespace(line[idx-1])
	nextOK := idx == len(line)-1 || isCaddyWhitespace(line[idx+1])
	return prevOK && nextOK
}

func isCaddyWhitespace(ch byte) bool {
	return ch == ' ' || ch == '\t' || ch == '\r' || ch == '\n'
}

func inferCaddyModulesFromConfig(config string) string {
	blocks, lines, err := parseTopLevelCaddyBlocks(config)
	if err != nil {
		return emptyModulesJSON
	}
	model := caddyFormModel{
		SchemaVersion: 1,
		Global:        caddyGlobal{Raw: extractCaddyGlobalRaw(blocks, lines)},
		Upstreams:     []caddyUpstream{},
		Sites:         []caddySite{},
	}
	for _, block := range blocks {
		if block.Kind != "site" {
			continue
		}
		site := inferCaddySiteFromBlock(lines, block)
		if len(site.Domains) > 0 {
			model.Sites = append(model.Sites, site)
		}
	}
	raw, err := json.Marshal(model)
	if err != nil {
		return emptyModulesJSON
	}
	return string(raw)
}

func extractCaddyGlobalRaw(blocks []caddyTopLevelBlock, lines []string) string {
	result := make([]string, 0)
	for _, block := range blocks {
		if block.Kind == "site" {
			continue
		}
		result = append(result, strings.TrimSpace(joinBlockLines(lines, block)))
	}
	return strings.TrimSpace(strings.Join(result, "\n\n"))
}

func inferCaddySiteFromBlock(lines []string, block caddyTopLevelBlock) caddySite {
	site := caddySite{
		ID:      fmt.Sprintf("site-%d", block.StartLine),
		Name:    strings.TrimSpace(block.Address),
		Enabled: true,
		Domains: strings.Fields(strings.TrimSpace(block.Address)),
		TLS:     &caddyTLS{Mode: "auto"},
		Imports: []string{},
		Routes:  []caddyRoute{},
	}
	depth := 0
	var currentRoute *caddyRoute
	handleDepth := -1
	currentRoot := ""

	for idx := block.StartLine + 1; idx < block.EndLine && idx < len(lines); idx++ {
		line := strings.TrimSpace(stripCaddyComment(strings.TrimRight(lines[idx], "\r\n")))
		if line == "" {
			continue
		}
		openCount, closeCount := countCaddyLineBraces(line)
		if handleDepth >= 0 && depth >= handleDepth && currentRoute != nil {
			inferCaddyHandleLine(currentRoute, line, currentRoot)
			if strings.HasPrefix(line, "root ") {
				currentRoot = parseCaddyRoot(line)
			}
			depth += openCount - closeCount
			if depth < handleDepth {
				currentRoute = nil
				handleDepth = -1
				currentRoot = ""
			}
			continue
		}

		if depth == 0 {
			switch {
			case strings.HasPrefix(line, "import "):
				if value := strings.TrimSpace(strings.TrimPrefix(line, "import ")); value != "" {
					site.Imports = append(site.Imports, value)
				}
			case strings.HasPrefix(line, "encode "):
				site.Encode = strings.Fields(strings.TrimSpace(strings.TrimPrefix(line, "encode ")))
			case strings.HasPrefix(line, "tls") && !strings.HasPrefix(line, "tls_insecure"):
				site.TLS = inferCaddyTLS(line)
			case strings.HasPrefix(line, "root "):
				currentRoot = parseCaddyRoot(line)
			case strings.HasPrefix(line, "handle"):
				route := inferCaddyRouteFromHandleLine(line, idx)
				site.Routes = append(site.Routes, route)
				currentRoute = &site.Routes[len(site.Routes)-1]
				handleDepth = depth + openCount
			default:
				route := ensureInferDefaultRoute(&site, idx)
				inferCaddyHandleLine(route, line, currentRoot)
			}
		}
		depth += openCount - closeCount
		if depth < 0 {
			depth = 0
		}
	}
	return site
}

func inferCaddyRouteFromHandleLine(line string, idx int) caddyRoute {
	name := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "handle"), "{"))
	match := caddyMatch{}
	if strings.HasPrefix(name, "/") || strings.HasPrefix(name, "*") {
		match.Path = []string{name}
	}
	if name == "" {
		name = "默认路由"
	}
	return caddyRoute{
		ID:      fmt.Sprintf("route-%d", idx),
		Name:    name,
		Enabled: true,
		Match:   match,
		Handles: []caddyHandle{},
	}
}

func ensureInferDefaultRoute(site *caddySite, idx int) *caddyRoute {
	if len(site.Routes) == 0 {
		site.Routes = append(site.Routes, caddyRoute{
			ID:      fmt.Sprintf("route-%d", idx),
			Name:    "默认路由",
			Enabled: true,
			Match:   caddyMatch{},
			Handles: []caddyHandle{},
		})
	}
	return &site.Routes[len(site.Routes)-1]
}

func inferCaddyHandleLine(route *caddyRoute, line, currentRoot string) {
	if route == nil {
		return
	}
	switch {
	case strings.HasPrefix(line, "reverse_proxy "):
		route.Handles = append(route.Handles, caddyHandle{
			ID:       fmt.Sprintf("%s-h%d", route.ID, len(route.Handles)+1),
			Type:     "reverse_proxy",
			Enabled:  true,
			Upstream: parseCaddyDirectiveArgs(line, "reverse_proxy"),
		})
	case strings.HasPrefix(line, "file_server"):
		route.Handles = append(route.Handles, caddyHandle{
			ID:      fmt.Sprintf("%s-h%d", route.ID, len(route.Handles)+1),
			Type:    "file_server",
			Enabled: true,
			Root:    currentRoot,
			Browse:  strings.Contains(line, " browse"),
		})
	case strings.HasPrefix(line, "respond "):
		body, status := parseCaddyRespond(line)
		route.Handles = append(route.Handles, caddyHandle{
			ID:      fmt.Sprintf("%s-h%d", route.ID, len(route.Handles)+1),
			Type:    "respond",
			Enabled: true,
			Body:    body,
			Status:  status,
		})
	case strings.HasPrefix(line, "redir "):
		to, code := parseCaddyRedirect(line)
		route.Handles = append(route.Handles, caddyHandle{
			ID:      fmt.Sprintf("%s-h%d", route.ID, len(route.Handles)+1),
			Type:    "redirect",
			Enabled: true,
			To:      to,
			Code:    code,
		})
	case strings.HasPrefix(line, "rewrite "):
		route.Handles = append(route.Handles, caddyHandle{
			ID:      fmt.Sprintf("%s-h%d", route.ID, len(route.Handles)+1),
			Type:    "rewrite",
			Enabled: true,
			URI:     parseCaddyRewrite(line),
		})
	case strings.HasPrefix(line, "header "):
		route.Handles = append(route.Handles, caddyHandle{
			ID:      fmt.Sprintf("%s-h%d", route.ID, len(route.Handles)+1),
			Type:    "header",
			Enabled: true,
			Rules:   []caddyHeaderRule{parseCaddyHeader(line)},
		})
	case strings.HasPrefix(line, "log_append "):
		parts := strings.Fields(strings.TrimPrefix(line, "log_append "))
		if len(parts) > 0 {
			route.LogAppend = append(route.LogAppend, caddyKV{Key: parts[0], Value: strings.Join(parts[1:], " ")})
		}
	}
}

func inferCaddyTLS(line string) *caddyTLS {
	parts := strings.Fields(line)
	if len(parts) <= 1 {
		return &caddyTLS{Mode: "auto"}
	}
	switch parts[1] {
	case "off":
		return &caddyTLS{Mode: "off"}
	case "internal":
		return &caddyTLS{Mode: "internal"}
	default:
		if len(parts) >= 3 {
			return &caddyTLS{Mode: "manual", CertFile: parts[1], KeyFile: parts[2]}
		}
	}
	return &caddyTLS{Mode: "auto"}
}

func parseCaddyRoot(line string) string {
	parts := strings.Fields(line)
	if len(parts) < 3 {
		return ""
	}
	idx := 1
	if parts[idx] == "*" || strings.HasPrefix(parts[idx], "@") {
		idx++
	}
	return strings.Join(parts[idx:], " ")
}

func parseCaddyDirectiveArgs(line, directive string) string {
	rest := strings.TrimSpace(strings.TrimPrefix(line, directive))
	rest = strings.TrimSuffix(rest, "{")
	return strings.TrimSpace(rest)
}

func parseCaddyRespond(line string) (string, int) {
	rest := strings.TrimSpace(strings.TrimPrefix(line, "respond "))
	status := 200
	parts := strings.Fields(rest)
	if len(parts) > 0 {
		if parsed, err := strconv.Atoi(parts[len(parts)-1]); err == nil {
			status = parsed
			rest = strings.TrimSpace(strings.TrimSuffix(rest, parts[len(parts)-1]))
		}
	}
	return strings.Trim(rest, `"`), status
}

func parseCaddyRedirect(line string) (string, int) {
	parts := strings.Fields(strings.TrimPrefix(line, "redir "))
	if len(parts) == 0 {
		return "/", 302
	}
	code := 302
	if len(parts) > 1 {
		switch parts[1] {
		case "permanent":
			code = 308
		case "temporary":
			code = 302
		default:
			if parsed, err := strconv.Atoi(parts[1]); err == nil {
				code = parsed
			}
		}
	}
	return parts[0], code
}

func parseCaddyRewrite(line string) string {
	parts := strings.Fields(strings.TrimPrefix(line, "rewrite "))
	if len(parts) >= 2 {
		return parts[1]
	}
	if len(parts) == 1 {
		return parts[0]
	}
	return "/"
}

func parseCaddyHeader(line string) caddyHeaderRule {
	rest := strings.TrimSpace(strings.TrimPrefix(line, "header "))
	op := "set"
	if strings.HasPrefix(rest, "-") {
		op = "delete"
		rest = strings.TrimPrefix(rest, "-")
	}
	parts := strings.Fields(rest)
	if len(parts) == 0 {
		return caddyHeaderRule{Op: op}
	}
	return caddyHeaderRule{Op: op, Key: parts[0], Value: strings.Join(parts[1:], " ")}
}

func enabledCaddyHandles(handles []caddyHandle) []caddyHandle {
	result := make([]caddyHandle, 0, len(handles))
	for _, handle := range handles {
		if handle.Enabled {
			result = append(result, handle)
		}
	}
	return result
}

func handlesAllType(handles []caddyHandle, expected string) bool {
	for _, handle := range handles {
		if handle.Type != expected {
			return false
		}
	}
	return len(handles) > 0
}

func trimStringSlice(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func isValidCaddySiteAddress(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	if caddyPortOnlyPattern.MatchString(value) || caddyDomainPattern.MatchString(value) {
		return true
	}
	host, port, ok := strings.Cut(value, ":")
	if ok {
		if _, err := strconv.Atoi(port); err != nil {
			return false
		}
		return host == "localhost" || caddyDomainPattern.MatchString(host)
	}
	return value == "localhost"
}

func isValidCaddyPathPattern(value string) bool {
	value = strings.TrimSpace(value)
	return strings.HasPrefix(value, "/") || strings.HasPrefix(value, "*") || (strings.HasPrefix(value, "{") && strings.HasSuffix(value, "}"))
}

func browseSuffix(browse bool) string {
	if browse {
		return " browse"
	}
	return ""
}

func caddyRedirectCode(code int) string {
	switch code {
	case 308:
		return "permanent"
	case 302:
		return "temporary"
	default:
		return strconv.Itoa(code)
	}
}

func caddyConfigFromServer(server *caddymodel.CaddyServer) (string, string) {
	if server == nil {
		return "", emptyModulesJSON
	}
	if strings.TrimSpace(server.Config) == "" {
		return "", emptyModulesJSON
	}
	return server.Config, normalizeCaddyModulesJSON(server.Modules)
}

func resolveCaddyConfigModulesForUpdate(mode, requestModules string) string {
	normalizedMode := normalizeCaddyConfigMode(mode, requestModules)
	if normalizedMode == caddyConfigModeRaw {
		return emptyModulesJSON
	}
	return normalizeCaddyModulesJSON(requestModules)
}
