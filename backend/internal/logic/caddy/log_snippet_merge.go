package caddy

import (
	"fmt"
	"strings"
)

// caddyLogSnippetName 是 LogFlux 托管的站点日志 snippet 名，站点侧以 import 引用。
const caddyLogSnippetName = "subdomain-log"

// caddyLogSnippetHead 是 snippet 定义块头前缀，完整形态为 "(subdomain-log) {"。
const caddyLogSnippetHead = "(" + caddyLogSnippetName + ")"

// isCaddyLogSnippetHead 判断一行（已去空白）是否为该 snippet 的块头。
//
// 前缀匹配后还要紧跟 "{"，避免把 (subdomain-logger) 之类的同名前缀块误判。
func isCaddyLogSnippetHead(trimmed string) bool {
	if !strings.HasPrefix(trimmed, caddyLogSnippetHead) {
		return false
	}
	rest := strings.TrimSpace(strings.TrimPrefix(trimmed, caddyLogSnippetHead))
	return strings.HasPrefix(rest, "{")
}

// braceDelta 统计一行里的花括号净增量，空行与整行注释不参与计数。
func braceDelta(line string) int {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return 0
	}
	return strings.Count(trimmed, "{") - strings.Count(trimmed, "}")
}

// findLogSnippetRange 定位配置中 (subdomain-log) 定义块的首尾行下标（含块头与收尾）。
//
// 块头之后按花括号深度配对，遇到深度回到 0 的顶层 "}" 收尾。未找到返回 -1, -1；
// 块头存在但始终没有收尾时返回 start, -1，由调用方按数据错误处理。
func findLogSnippetRange(lines []string) (int, int) {
	start := -1
	depth := 0
	for idx, line := range lines {
		if start < 0 {
			if isCaddyLogSnippetHead(strings.TrimSpace(line)) {
				start = idx
				depth = braceDelta(line)
			}
			continue
		}
		depth += braceDelta(line)
		if depth <= 0 {
			return start, idx
		}
	}
	if start >= 0 {
		return start, -1
	}
	return -1, -1
}

// findGlobalOptionsRange 定位配置的全局选项块（顶层独立成行的 "{" 到配对的 "}"）。
//
// 未找到返回 -1, -1。全局块只可能在顶层，缩进的 "{" 属于某个块内部，不算。
func findGlobalOptionsRange(lines []string) (int, int) {
	start := -1
	depth := 0
	for idx, line := range lines {
		if start < 0 {
			if strings.TrimSpace(line) != "{" {
				continue
			}
			if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
				continue
			}
			start = idx
			depth = 1
			continue
		}
		depth += braceDelta(line)
		if depth <= 0 {
			return start, idx
		}
	}
	return -1, -1
}

// findFirstSiteBlockStart 返回第一个站点块块头所在行；没有站点块时返回 -1。
//
// 只看顶层的块头行（含 "{"），跳过顶层单行指令、其他 snippet 定义与已配对的 "}"。
func findFirstSiteBlockStart(lines []string) int {
	for idx, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || trimmed == "}" || trimmed == "{" {
			continue
		}
		if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
			continue
		}
		if !strings.Contains(trimmed, "{") {
			continue
		}
		// 其他 snippet 定义（(waf_protect) { 等）不算站点块。
		if strings.HasPrefix(trimmed, "(") {
			continue
		}
		return idx
	}
	return -1
}

// logSnippetInsertIndex 计算 snippet 的插入行下标：全局选项块之后、第一个站点块之前。
//
// 全局块存在时紧跟其后（吞掉其后的空行，由调用方补一个分隔空行）；否则插到第一个
// 站点块之前，并把紧贴该块的注释行一起留在原位，保证注释仍附着在所属站点块上。
func logSnippetInsertIndex(lines []string) int {
	if _, end := findGlobalOptionsRange(lines); end >= 0 {
		at := end + 1
		for at < len(lines) && strings.TrimSpace(lines[at]) == "" {
			at++
		}
		return at
	}
	start := findFirstSiteBlockStart(lines)
	if start < 0 {
		return len(lines)
	}
	for start > 0 {
		prev := strings.TrimSpace(lines[start-1])
		if prev == "" || !strings.HasPrefix(prev, "#") {
			break
		}
		start--
	}
	return start
}

// logSnippetHeaderStart 从块头向上扩展，把紧贴块头的连续注释行算进替换范围。
//
// 注释紧贴块头时视为该定义的一部分（与插入时"注释留在所属块"同一规则）；
// 不这样做的话，每次应用都会把定义自带的注释再叠一份上去，配置越合越长。
func logSnippetHeaderStart(lines []string, start int) int {
	for start > 0 {
		prev := strings.TrimSpace(lines[start-1])
		if prev == "" || !strings.HasPrefix(prev, "#") {
			break
		}
		start--
	}
	return start
}

// mergeCaddyLogSnippet 把日志 snippet 定义合并进配置。
//
// 已有定义（连同紧贴块头的注释）整体替换，没有则插到全局选项块之后、第一个站点块
// 之前；两者都没有时追加到文件末尾。插入与替换都不打散其余字节，缩进、注释原样
// 保留，换行风格跟随原文。
func mergeCaddyLogSnippet(config, snippet string) (string, error) {
	snippet = strings.TrimSpace(strings.ReplaceAll(snippet, "\r\n", "\n"))
	if snippet == "" {
		return config, nil
	}
	if strings.TrimSpace(config) == "" {
		return snippet, nil
	}
	snippetLines := strings.Split(snippet, "\n")

	headFound := false
	for _, line := range snippetLines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if !isCaddyLogSnippetHead(trimmed) {
			return "", fmt.Errorf("日志 snippet 块头必须是 %s {", caddyLogSnippetHead)
		}
		headFound = true
		break
	}
	if !headFound {
		return "", fmt.Errorf("日志 snippet 内容为空")
	}
	total := 0
	for _, line := range snippetLines {
		total += braceDelta(line)
	}
	if total != 0 {
		return "", fmt.Errorf("日志 snippet 花括号不配对")
	}

	normalized := strings.ReplaceAll(config, "\r\n", "\n")
	lines := strings.Split(normalized, "\n")

	out := make([]string, 0, len(lines)+len(snippetLines)+2)
	start, end := findLogSnippetRange(lines)
	switch {
	case start >= 0 && end < 0:
		return "", fmt.Errorf("配置中的 %s 块缺少收尾 }", caddyLogSnippetHead)
	case start >= 0:
		out = append(out, lines[:logSnippetHeaderStart(lines, start)]...)
		out = append(out, snippetLines...)
		out = append(out, lines[end+1:]...)
	default:
		at := logSnippetInsertIndex(lines)
		out = append(out, lines[:at]...)
		if len(out) > 0 && strings.TrimSpace(out[len(out)-1]) != "" {
			out = append(out, "")
		}
		out = append(out, snippetLines...)
		if at < len(lines) {
			out = append(out, "")
			out = append(out, lines[at:]...)
		}
	}

	result := strings.Join(out, "\n")
	if strings.Contains(config, "\r\n") {
		result = strings.ReplaceAll(result, "\n", "\r\n")
	}
	return result, nil
}
