package caddy

import (
	"fmt"
	"strings"
)

const wafProtectImportLine = "import waf_protect"

type caddyTopLevelBlock struct {
	Kind      string
	Name      string
	Address   string
	Header    string
	StartLine int
	EndLine   int
}

type wafIntegrationSnapshot struct {
	OrderReady     bool
	SnippetReady   bool
	DirectiveReady bool
	ImportedSites  []string
	AvailableSites []string
}

func inspectWafIntegration(config string) (*wafIntegrationSnapshot, error) {
	blocks, lines, err := parseTopLevelCaddyBlocks(config)
	if err != nil {
		return nil, err
	}

	snapshot := &wafIntegrationSnapshot{
		ImportedSites:  make([]string, 0),
		AvailableSites: make([]string, 0),
	}

	for _, block := range blocks {
		switch block.Kind {
		case "options":
			if blockContainsExactLine(lines, block, "order coraza_waf first") {
				snapshot.OrderReady = true
			}
		case "snippet":
			if block.Name != "waf_protect" {
				continue
			}
			content := joinBlockLines(lines, block)
			if strings.Contains(content, "coraza_waf") {
				snapshot.SnippetReady = true
			}
			if strings.Contains(content, "directives `") {
				snapshot.DirectiveReady = true
			}
		case "site":
			address := strings.TrimSpace(block.Address)
			if address == "" {
				continue
			}
			snapshot.AvailableSites = append(snapshot.AvailableSites, address)
			if blockContainsExactLine(lines, block, wafProtectImportLine) {
				snapshot.ImportedSites = append(snapshot.ImportedSites, address)
			}
		}
	}

	return snapshot, nil
}

func buildWafIntegrationStatusMessage(snapshot *wafIntegrationSnapshot) string {
	if snapshot == nil {
		return "未获取到 WAF 接入状态"
	}
	if len(snapshot.AvailableSites) == 0 {
		return "未识别到可接入的站点块，请先保存完整 Caddy 配置"
	}
	if !snapshot.OrderReady {
		return "尚未注入全局 order coraza_waf first"
	}
	if !snapshot.SnippetReady {
		return "尚未注入 waf_protect 片段"
	}
	if !snapshot.DirectiveReady {
		return "waf_protect 片段缺少 Coraza directives 配置块"
	}
	if len(snapshot.ImportedSites) == 0 {
		return "已预埋 Coraza 配置，尚未挂载到任何站点"
	}
	return "已完成 Coraza 接入，可继续通过策略模式切换 Off / DetectionOnly / On"
}

func ensureCorazaOrder(config string) (string, bool, error) {
	blocks, lines, err := parseTopLevelCaddyBlocks(config)
	if err != nil {
		return "", false, err
	}
	newline := detectCaddyNewline(config)

	for _, block := range blocks {
		if block.Kind != "options" {
			continue
		}
		if blockContainsExactLine(lines, block, "order coraza_waf first") {
			return config, false, nil
		}
		indent := detectBlockChildIndent(lines, block, "  ")
		insertLine := indent + "order coraza_waf first" + newline
		updated := insertLines(lines, block.StartLine+1, []string{insertLine})
		return strings.Join(updated, ""), true, nil
	}

	prefix := []string{
		"{" + newline,
		"  order coraza_waf first" + newline,
		"}" + newline,
	}
	if strings.TrimSpace(config) != "" {
		prefix = append(prefix, newline)
	}
	updated := append(prefix, lines...)
	return strings.Join(updated, ""), true, nil
}

func ensureWafProtectSnippet(config string) (string, bool, error) {
	blocks, lines, err := parseTopLevelCaddyBlocks(config)
	if err != nil {
		return "", false, err
	}
	newline := detectCaddyNewline(config)
	templateLines := splitLinesKeepEndings(renderWafProtectSnippet(newline))

	for _, block := range blocks {
		if block.Kind != "snippet" || block.Name != "waf_protect" {
			continue
		}
		content := joinBlockLines(lines, block)
		if strings.Contains(content, "coraza_waf") && strings.Contains(content, "directives `") {
			return config, false, nil
		}
		updated := replaceLineRange(lines, block.StartLine, block.EndLine, templateLines)
		return strings.Join(updated, ""), true, nil
	}

	insertAt := len(lines)
	for _, block := range blocks {
		if block.Kind == "site" {
			insertAt = block.StartLine
			break
		}
	}
	insertPayload := append([]string{}, templateLines...)
	if insertAt < len(lines) {
		insertPayload = append(insertPayload, newline)
	}
	updated := insertLines(lines, insertAt, insertPayload)
	return strings.Join(updated, ""), true, nil
}

func ensureSiteImport(config, siteAddress string) (string, bool, error) {
	block, lines, err := findSiteBlock(config, siteAddress)
	if err != nil {
		return "", false, err
	}
	if blockContainsExactLine(lines, *block, wafProtectImportLine) {
		return config, false, nil
	}
	indent := detectBlockChildIndent(lines, *block, "  ")
	newline := detectCaddyNewline(config)
	updated := insertLines(lines, block.StartLine+1, []string{indent + wafProtectImportLine + newline})
	return strings.Join(updated, ""), true, nil
}

func removeSiteImport(config, siteAddress string) (string, bool, error) {
	block, lines, err := findSiteBlock(config, siteAddress)
	if err != nil {
		return "", false, err
	}

	removed := false
	nextLines := make([]string, 0, len(lines))
	for idx, line := range lines {
		if idx > block.StartLine && idx <= block.EndLine && strings.TrimSpace(strings.TrimRight(line, "\r\n")) == wafProtectImportLine {
			removed = true
			continue
		}
		nextLines = append(nextLines, line)
	}
	if !removed {
		return config, false, nil
	}
	return strings.Join(nextLines, ""), true, nil
}

func findSiteBlock(config, siteAddress string) (*caddyTopLevelBlock, []string, error) {
	blocks, lines, err := parseTopLevelCaddyBlocks(config)
	if err != nil {
		return nil, nil, err
	}
	target := strings.TrimSpace(siteAddress)
	for _, block := range blocks {
		if block.Kind == "site" && strings.TrimSpace(block.Address) == target {
			copied := block
			return &copied, lines, nil
		}
	}
	return nil, nil, fmt.Errorf("站点不存在: %s", target)
}

func parseTopLevelCaddyBlocks(config string) ([]caddyTopLevelBlock, []string, error) {
	lines := splitLinesKeepEndings(config)
	blocks := make([]caddyTopLevelBlock, 0)
	depth := 0
	inBacktick := false
	current := (*caddyTopLevelBlock)(nil)

	for idx, line := range lines {
		trimmed := strings.TrimSpace(strings.TrimRight(line, "\r\n"))
		if depth == 0 && !inBacktick && current == nil && isTopLevelBlockHeader(trimmed) {
			current = &caddyTopLevelBlock{
				Header:    trimmed,
				StartLine: idx,
			}
			classifyTopLevelBlock(current)
		}

		updateCaddyParseState(line, &depth, &inBacktick)

		if current != nil && depth == 0 && !inBacktick {
			current.EndLine = idx
			blocks = append(blocks, *current)
			current = nil
		}
	}

	if current != nil || depth != 0 || inBacktick {
		return nil, nil, fmt.Errorf("Caddy 配置结构无效")
	}
	return blocks, lines, nil
}

func isTopLevelBlockHeader(trimmed string) bool {
	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return false
	}
	return strings.HasSuffix(trimmed, "{")
}

func classifyTopLevelBlock(block *caddyTopLevelBlock) {
	if block == nil {
		return
	}
	header := strings.TrimSpace(block.Header)
	switch {
	case header == "{":
		block.Kind = "options"
	case strings.HasPrefix(header, "("):
		block.Kind = "snippet"
		if end := strings.Index(header, ")"); end > 1 {
			block.Name = strings.TrimSpace(header[1:end])
		}
	default:
		block.Kind = "site"
		block.Address = strings.TrimSpace(strings.TrimSuffix(header, "{"))
	}
}

func updateCaddyParseState(line string, depth *int, inBacktick *bool) {
	if depth == nil || inBacktick == nil {
		return
	}
	for idx := 0; idx < len(line); idx++ {
		ch := line[idx]
		if *inBacktick {
			if ch == '`' {
				*inBacktick = false
			}
			continue
		}
		if ch == '#' {
			break
		}
		if ch == '`' {
			*inBacktick = true
			continue
		}
		switch ch {
		case '{':
			*depth = *depth + 1
		case '}':
			if *depth > 0 {
				*depth = *depth - 1
			}
		}
	}
}

func blockContainsExactLine(lines []string, block caddyTopLevelBlock, target string) bool {
	for idx := block.StartLine + 1; idx < block.EndLine; idx++ {
		if strings.TrimSpace(strings.TrimRight(lines[idx], "\r\n")) == target {
			return true
		}
	}
	return false
}

func joinBlockLines(lines []string, block caddyTopLevelBlock) string {
	if block.StartLine < 0 || block.EndLine >= len(lines) || block.StartLine > block.EndLine {
		return ""
	}
	return strings.Join(lines[block.StartLine:block.EndLine+1], "")
}

func detectBlockChildIndent(lines []string, block caddyTopLevelBlock, fallback string) string {
	for idx := block.StartLine + 1; idx < block.EndLine; idx++ {
		trimmed := strings.TrimSpace(strings.TrimRight(lines[idx], "\r\n"))
		if trimmed == "" || trimmed == "}" {
			continue
		}
		return leadingWhitespace(lines[idx])
	}
	if fallback != "" {
		return fallback
	}
	return "  "
}

func detectCaddyNewline(config string) string {
	if strings.Contains(config, "\r\n") {
		return "\r\n"
	}
	return "\n"
}

func splitLinesKeepEndings(config string) []string {
	if config == "" {
		return []string{}
	}
	parts := strings.SplitAfter(config, "\n")
	if len(parts) > 0 && parts[len(parts)-1] == "" {
		return parts[:len(parts)-1]
	}
	return parts
}

func insertLines(lines []string, index int, insert []string) []string {
	if index < 0 {
		index = 0
	}
	if index > len(lines) {
		index = len(lines)
	}
	updated := make([]string, 0, len(lines)+len(insert))
	updated = append(updated, lines[:index]...)
	updated = append(updated, insert...)
	updated = append(updated, lines[index:]...)
	return updated
}

func replaceLineRange(lines []string, start, end int, replacement []string) []string {
	if start < 0 {
		start = 0
	}
	if end >= len(lines) {
		end = len(lines) - 1
	}
	updated := make([]string, 0, len(lines)-(end-start+1)+len(replacement))
	updated = append(updated, lines[:start]...)
	updated = append(updated, replacement...)
	updated = append(updated, lines[end+1:]...)
	return updated
}

func renderWafProtectSnippet(newline string) string {
	return strings.Join([]string{
		"(waf_protect) {",
		"  coraza_waf {",
		"    load_owasp_crs",
		"    directives `",
		"      Include @coraza.conf-recommended",
		"      Include @crs-setup.conf.example",
		"      Include @owasp_crs/*.conf",
		"",
		"      SecRuleEngine Off",
		"      SecAuditEngine RelevantOnly",
		"      SecAuditLogFormat JSON",
		"      SecAuditLog /var/log/caddy/waf_audit.log",
		"",
		"      SecRequestBodyAccess On",
		"      SecRequestBodyLimit 10485760",
		"      SecRequestBodyNoFilesLimit 1048576",
		"    `",
		"  }",
		"}",
	}, newline) + newline
}
