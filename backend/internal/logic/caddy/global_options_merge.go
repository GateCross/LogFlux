package caddy

import (
	"strings"
)

// caddyGlobalOptionsKeys 是全局选项合并接管的顶层指令名集合。
//
// trusted_proxies 不是全局块顶层指令（Caddy 要求嵌在 servers {} 子块里），
// 单独走 mergeTrustedProxies 的块内手术，不在此列。
var caddyGlobalOptionsKeys = []string{"auto_https", "debug", "email"}

// buildGlobalOptionsLines 把已配置的全局选项渲染为全局块内的顶层指令行。
//
// 只输出显式配置了的键；extraRawOptions 是逃生舱，逐行原样输出。
// trusted_proxies 不在此渲染（见 mergeTrustedProxies）。
func buildGlobalOptionsLines(opts CaddyGlobalOptionsValue) []string {
	var lines []string
	if v := strings.TrimSpace(opts.AutoHttps); v != "" {
		lines = append(lines, "  auto_https "+v)
	}
	if opts.Debug != nil && *opts.Debug {
		lines = append(lines, "  debug")
	}
	if v := strings.TrimSpace(opts.Email); v != "" {
		lines = append(lines, "  email "+v)
	}
	for _, raw := range strings.Split(opts.ExtraRawOptions, "\n") {
		if trimmed := strings.TrimSpace(raw); trimmed != "" {
			lines = append(lines, "  "+trimmed)
		}
	}
	return lines
}

// renderTrustedProxiesLine 渲染 servers 子块内的 trusted_proxies 行。
func renderTrustedProxiesLine(opts CaddyGlobalOptionsValue) string {
	cidrs := make([]string, 0, len(opts.TrustedProxies))
	for _, item := range opts.TrustedProxies {
		if trimmed := strings.TrimSpace(item); trimmed != "" {
			cidrs = append(cidrs, trimmed)
		}
	}
	if len(cidrs) == 0 {
		return ""
	}
	return "trusted_proxies static " + strings.Join(cidrs, " ")
}

// globalOptionsOwnedKeys 返回本次设置里显式接管了的顶层指令键集合。
func globalOptionsOwnedKeys(opts CaddyGlobalOptionsValue) map[string]bool {
	owned := map[string]bool{}
	if strings.TrimSpace(opts.AutoHttps) != "" {
		owned["auto_https"] = true
	}
	if opts.Debug != nil {
		owned["debug"] = true
	}
	if strings.TrimSpace(opts.Email) != "" {
		owned["email"] = true
	}
	if strings.TrimSpace(opts.ExtraRawOptions) != "" {
		for _, raw := range strings.Split(opts.ExtraRawOptions, "\n") {
			if trimmed := strings.TrimSpace(raw); trimmed != "" {
				owned[trimmed] = true
			}
		}
	}
	return owned
}

// splitGlobalDirective 判断一行全局块内指令是否以 name 开头（注释行不算）。
func isGlobalDirectiveLine(line, name string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return false
	}
	return trimmed == name || strings.HasPrefix(trimmed, name+" ")
}

// mergeGlobalOptions 把全局选项合并进配置的首个全局块。
//
// 冲突策略：全局设置为准——设置里显式配置了的键，块内同名指令删除后写入
// 设置值；未配置的键不输出、也不动块内既有内容（首次启用不会悄悄删掉
// 配置里的手写值）。没有全局块时在文件头生成一个。幂等。
func mergeGlobalOptions(config string, opts CaddyGlobalOptionsValue) string {
	desired := buildGlobalOptionsLines(opts)
	trustedLine := renderTrustedProxiesLine(opts)
	if strings.TrimSpace(config) == "" && len(desired) == 0 && trustedLine == "" {
		return config
	}
	ownedKeys := globalOptionsOwnedKeys(opts)
	owned := func(line string) bool {
		trimmed := strings.TrimSpace(line)
		for _, key := range caddyGlobalOptionsKeys {
			if ownedKeys[key] && isGlobalDirectiveLine(line, key) {
				return true
			}
		}
		// trusted_proxies 由 mergeTrustedProxies 原位处理，这里只剔除
		// 历史遗留的顶层写法（非法 Caddyfile，接管时顺手清掉）。
		if trustedLine != "" && isGlobalDirectiveLine(line, "trusted_proxies") {
			return true
		}
		return ownedKeys[trimmed]
	}

	normalized := strings.ReplaceAll(config, "\r\n", "\n")
	lines := strings.Split(normalized, "\n")

	// 定位首个全局块（顶层独立成行的 "{" 与 "}"）。
	start, end := -1, -1
	depth := 0
	for idx, line := range lines {
		trimmed := strings.TrimSpace(line)
		if depth == 0 && trimmed == "{" && start < 0 {
			start = idx
		}
		if start >= 0 {
			for _, ch := range trimmed {
				if ch == '{' {
					depth++
				} else if ch == '}' {
					depth--
				}
			}
			if depth == 0 && start >= 0 && idx > start {
				end = idx
				break
			}
		}
	}

	var out []string
	if start < 0 {
		// 无全局块：仅在确有选项时生成，置于文件头。
		if len(desired) == 0 && trustedLine == "" {
			return config
		}
		out = append(out, "{")
		out = append(out, desired...)
		if trustedLine != "" {
			out = append(out, "  servers {", "    "+trustedLine, "  }")
		}
		out = append(out, "}")
		out = append(out, lines...)
	} else {
		// 全局块内：先剔除被接管的顶层行，再把期望行插到块头之后。
		var inner []string
		for idx := start + 1; idx < end; idx++ {
			if !owned(lines[idx]) {
				inner = append(inner, lines[idx])
			}
		}
		merged := append([]string{}, desired...)
		if trustedLine != "" {
			inner = mergeTrustedProxies(inner, trustedLine)
		}
		merged = append(merged, inner...)
		out = append(out, lines[:start]...)
		out = append(out, "{")
		out = append(out, merged...)
		out = append(out, "}")
		out = append(out, lines[end+1:]...)
	}

	result := strings.Join(out, "\n")
	if strings.HasSuffix(config, "\r\n") || strings.Contains(config, "\r\n") {
		result = strings.ReplaceAll(result, "\n", "\r\n")
	}
	return result
}

// mergeTrustedProxies 在全局块内层行里原位处理 trusted_proxies。
//
// 有 servers 子块：替换块内既有 trusted_proxies 行；没有则插到块头，
// 保留块内其他子指令。没有 servers 子块：在末尾追加一个。
func mergeTrustedProxies(inner []string, trustedLine string) []string {
	// 定位 servers 子块（块名独立成行的 "servers {"）。
	serversStart := -1
	for idx, line := range linesTrimmed(inner) {
		if line == "servers {" || line == "servers{" {
			serversStart = idx
			break
		}
	}
	if serversStart < 0 {
		return append(inner, "  servers {", "    "+trustedLine, "  }")
	}

	// servers 块内：替换第一个 trusted_proxies，删除其余。
	depth := 1
	out := make([]string, 0, len(inner)+2)
	inserted := false
	for idx := 0; idx < len(inner); idx++ {
		line := inner[idx]
		trimmed := strings.TrimSpace(line)
		if idx > serversStart && depth > 0 {
			for _, ch := range trimmed {
				if ch == '{' {
					depth++
				} else if ch == '}' {
					depth--
				}
			}
			if depth == 0 {
				out = append(out, line)
				continue
			}
			if isGlobalDirectiveLine(line, "trusted_proxies") {
				if !inserted {
					indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
					out = append(out, indent+trustedLine)
					inserted = true
				}
				continue
			}
		}
		out = append(out, line)
	}
	if !inserted {
		// 块内原本没有：插到 servers { 之后一行。
		withLine := make([]string, 0, len(out)+1)
		withLine = append(withLine, out[:serversStart+1]...)
		withLine = append(withLine, "    "+trustedLine)
		withLine = append(withLine, out[serversStart+1:]...)
		return withLine
	}
	return out
}

func linesTrimmed(lines []string) []string {
	out := make([]string, len(lines))
	for idx, line := range lines {
		out[idx] = strings.TrimSpace(line)
	}
	return out
}
