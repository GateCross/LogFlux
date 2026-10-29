package caddy

import (
	"encoding/json"
	"errors"
	"fmt"
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
	// 与前端 HEALTH_DURATION_RE 一致：正整数 + 单位 ms|s|m|h
	caddyHealthDurationPattern = regexp.MustCompile(`^[1-9]\d*(ms|s|m|h)$`)
	caddyMethodAllowSet        = map[string]struct{}{
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
	// 排除规则不再由保存流程改写用户配置。
	//
	// 原先这里无条件注入 /api/ 与 Host-IP 两条 ctl:ruleEngine=Off 规则，既绕过
	// WAF 又让用户在原文模式删不掉。现在排除规则统一由策略渲染
	// （composeManagedCorazaDirectives）产出，保存只做格式化和校验，所见即所存。
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
	if strings.TrimSpace(formatted) == "" || !hasCaddyConfigContent(formatted) {
		return nil, fmt.Errorf("Caddy 配置不能为空，请粘贴有效 Caddyfile")
	}
	// 原文模式只做格式化，不注入任何规则：用户写什么就存什么。
	actions := []string{"使用原始 Caddyfile 内容"}
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
		for _, target := range trimStringSlice(upstream.Targets) {
			if err := caddyUpstreamPathError(target); err != "" {
				errors = append(errors, fmt.Sprintf("上游 %s %s", name, err))
			}
		}
		label := fmt.Sprintf("上游 %s 健康检查", firstNonEmpty(name, "未命名"))
		errors = append(errors, validateCaddyHealthCheckFields(upstream.HealthCheck, label)...)
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
			label := fmt.Sprintf("路由 %s 健康检查", routeName)
			errors = append(errors, validateCaddyHealthCheckFields(handle.HealthCheck, label)...)
			if _, ok := upstreamNames[strings.TrimSpace(handle.Upstream)]; ok {
				continue
			}
			// 内联地址同样过不了带 path 的上游；池名已在 targets 校验过
			if err := caddyUpstreamPathError(handle.Upstream); err != "" {
				errors = append(errors, fmt.Sprintf("路由 %s %s", routeName, err))
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
	var pool *caddyUpstream
	if upstream, ok := upstreams[targets]; ok {
		pool = &upstream
		if len(trimStringSlice(upstream.Targets)) > 0 {
			targets = strings.Join(trimStringSlice(upstream.Targets), " ")
		}
	}
	if targets == "" {
		targets = "localhost:8080"
	}

	healthCheck := resolveEffectiveHealthCheck(handle.HealthCheck, pool)
	lbPolicy := resolveEffectiveLbPolicy(handle.LBPolicy, pool)
	transport := strings.TrimSpace(handle.TransportProtocol)
	if transport == "" && handle.TLSInsecureSkipVerify {
		transport = "http"
	}

	if reverseProxyNeedsBlock(healthCheck, lbPolicy, transport, handle.TLSInsecureSkipVerify) {
		*lines = append(*lines, "    reverse_proxy "+targets+" {")
		appendReverseProxyBlockLines(lines, "      ", healthCheck, lbPolicy, transport, handle.TLSInsecureSkipVerify)
		*lines = append(*lines, "    }")
		return
	}
	*lines = append(*lines, "    reverse_proxy "+targets)
}

// 与前端一致：path 非空才可写出 health 指令。
func hasEmittableHealthCheck(healthCheck *caddyHealthCheck) bool {
	return healthCheck != nil && strings.TrimSpace(healthCheck.Path) != ""
}

// handle 级有效健康检查优先于被引用 pool 级（Req 3.4）。
func resolveEffectiveHealthCheck(handleHC *caddyHealthCheck, pool *caddyUpstream) *caddyHealthCheck {
	if hasEmittableHealthCheck(handleHC) {
		return normalizeHealthCheck(handleHC)
	}
	if pool != nil && hasEmittableHealthCheck(pool.HealthCheck) {
		return normalizeHealthCheck(pool.HealthCheck)
	}
	return nil
}

func normalizeHealthCheck(hc *caddyHealthCheck) *caddyHealthCheck {
	if hc == nil {
		return nil
	}
	out := &caddyHealthCheck{Path: strings.TrimSpace(hc.Path)}
	if interval := strings.TrimSpace(hc.Interval); interval != "" {
		out.Interval = interval
	}
	if timeout := strings.TrimSpace(hc.Timeout); timeout != "" {
		out.Timeout = timeout
	}
	return out
}

// handle 级非空负载策略优先，否则用 pool 级。
func resolveEffectiveLbPolicy(handlePolicy string, pool *caddyUpstream) string {
	if policy := strings.TrimSpace(handlePolicy); policy != "" {
		return policy
	}
	if pool != nil {
		if policy := strings.TrimSpace(pool.LBPolicy); policy != "" {
			return policy
		}
	}
	return ""
}

func reverseProxyNeedsBlock(healthCheck *caddyHealthCheck, lbPolicy, transport string, tlsInsecureSkipVerify bool) bool {
	if strings.TrimSpace(lbPolicy) != "" {
		return true
	}
	if hasEmittableHealthCheck(healthCheck) {
		return true
	}
	if strings.TrimSpace(transport) != "" || tlsInsecureSkipVerify {
		return true
	}
	return false
}

// 写出 lb_policy / health_* / transport，顺序与前端一致。
func appendReverseProxyBlockLines(lines *[]string, indent string, healthCheck *caddyHealthCheck, lbPolicy, transport string, tlsInsecureSkipVerify bool) {
	if policy := strings.TrimSpace(lbPolicy); policy != "" {
		*lines = append(*lines, indent+"lb_policy "+policy)
	}
	if hasEmittableHealthCheck(healthCheck) {
		path := strings.TrimSpace(healthCheck.Path)
		*lines = append(*lines, indent+"health_uri "+path)
		if interval := strings.TrimSpace(healthCheck.Interval); interval != "" {
			*lines = append(*lines, indent+"health_interval "+interval)
		}
		if timeout := strings.TrimSpace(healthCheck.Timeout); timeout != "" {
			*lines = append(*lines, indent+"health_timeout "+timeout)
		}
	}
	if transport != "" {
		*lines = append(*lines, indent+"transport "+transport+" {")
		if tlsInsecureSkipVerify {
			*lines = append(*lines, indent+"  tls_insecure_skip_verify")
		}
		*lines = append(*lines, indent+"}")
	}
}

// 与前端健康检查字段校验同一契约（中文字段错误）。
func validateCaddyHealthCheckFields(healthCheck *caddyHealthCheck, fieldLabel string) []string {
	if healthCheck == nil {
		return nil
	}
	if strings.TrimSpace(fieldLabel) == "" {
		fieldLabel = "健康检查"
	}
	errors := make([]string, 0, 3)
	path := strings.TrimSpace(healthCheck.Path)
	if path != "" && !strings.HasPrefix(path, "/") {
		errors = append(errors, fieldLabel+"路径必须以 / 开头")
	}
	interval := strings.TrimSpace(healthCheck.Interval)
	if interval != "" && !caddyHealthDurationPattern.MatchString(interval) {
		errors = append(errors, fieldLabel+"间隔格式无效，应为正整数加单位（ms|s|m|h）")
	}
	timeout := strings.TrimSpace(healthCheck.Timeout)
	if timeout != "" && !caddyHealthDurationPattern.MatchString(timeout) {
		errors = append(errors, fieldLabel+"超时格式无效，应为正整数加单位（ms|s|m|h）")
	}
	return errors
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

func hasCaddyConfigContent(content string) bool {
	for _, line := range strings.Split(content, "\n") {
		if strings.TrimSpace(stripCaddyComment(line)) != "" {
			return true
		}
	}
	return false
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

// caddyUpstreamPathError 校验反向代理上游：Caddy 只接受 scheme + host + port，
// 带 path / query 的地址会在 /adapt 阶段报错，所以写配置前就在这里拦下。
// unix/ 与 unix+h2c/ 后面的绝对路径属于合法网络地址形式，不算路径。返回空串表示合法。
func caddyUpstreamPathError(value string) string {
	addr := strings.TrimSpace(value)
	if addr == "" || strings.HasPrefix(addr, "unix/") || strings.HasPrefix(addr, "unix+h2c/") {
		return ""
	}
	// 去掉 scheme 只看 host[:port]，其中出现 / 或 ? 即为路径
	host := addr
	if idx := strings.Index(host, "://"); idx >= 0 {
		host = host[idx+3:]
	}
	if strings.ContainsAny(host, "/?") {
		return fmt.Sprintf("上游地址不支持路径: %s（Caddy 只接受 scheme://host:port，前缀请用 handle_path 单独配置）", addr)
	}
	return ""
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

func resolveCaddyConfigModules(mode, requestModules string) string {
	normalizedMode := normalizeCaddyConfigMode(mode, requestModules)
	if normalizedMode == caddyConfigModeQuick {
		return normalizeCaddyModulesJSON(requestModules)
	}
	if strings.TrimSpace(requestModules) != "" {
		return normalizeCaddyModulesJSON(requestModules)
	}
	// raw 模式手改文本后旧 modules 必然过期，继续保留会让 blocks 模式
	// 用旧快照覆盖用户刚保存的修改。宁可留空，让前端回退到文本解析。
	return emptyModulesJSON
}
