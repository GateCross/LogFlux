package caddy

import (
	"fmt"
	"strings"
)

// caddyEntryBlockAddress 是系统入口块的站点地址。
const caddyEntryBlockAddress = ":80"

// findEntryBlockRange 定位配置中 :80 站点块的首尾行下标（含块头与收尾大括号）。
//
// 站点块头特征：顶层（缩进为 0）非注释行，且首个 token 为 :80。
// 未找到返回 -1, -1。
func findEntryBlockRange(lines []string) (int, int) {
	start := -1
	for idx, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "}") {
			if start < 0 {
				continue
			}
		}
		if start < 0 {
			if trimmed == caddyEntryBlockAddress || strings.HasPrefix(trimmed, caddyEntryBlockAddress+" ") || strings.HasPrefix(trimmed, caddyEntryBlockAddress+"{") {
				start = idx
			}
			continue
		}
		// 块内寻找收尾：顶层（无缩进）的 "}"。
		if trimmed == "}" && !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
			return start, idx
		}
	}
	return -1, -1
}

// extractCaddyEntryBlock 从配置中提取 :80 站点块原文（含块头与收尾大括号）。
//
// 未找到时返回错误，调用方（同步为模板）需要明确知道配置里没有入口块。
func extractCaddyEntryBlock(config string) (string, error) {
	lines := strings.Split(strings.ReplaceAll(config, "\r\n", "\n"), "\n")
	start, end := findEntryBlockRange(lines)
	if start < 0 {
		return "", fmt.Errorf("当前配置中没有 %s 站点块", caddyEntryBlockAddress)
	}
	return strings.TrimSpace(strings.Join(lines[start:end+1], "\n")), nil
}

// mergeEntryBlock 用模板整体替换配置中的 :80 站点块。
//
// 没有现成块时把模板追加到文件末尾（首个站点块之前亦可，追加更简单且不破坏
// 既有顺序）。幂等：模板与配置中的块一致时不产生变化。
func mergeEntryBlock(config, entryBlock string) (string, error) {
	entryBlock = strings.TrimSpace(entryBlock)
	if entryBlock == "" {
		return config, nil
	}
	if !strings.Contains(entryBlock, caddyEntryBlockAddress) {
		return "", fmt.Errorf("入口块模板缺少 %s 站点地址", caddyEntryBlockAddress)
	}

	normalized := strings.ReplaceAll(config, "\r\n", "\n")
	lines := strings.Split(normalized, "\n")
	start, end := findEntryBlockRange(lines)

	var out []string
	if start < 0 {
		out = append(out, lines...)
		if len(out) > 0 && strings.TrimSpace(out[len(out)-1]) != "" {
			out = append(out, "")
		}
		out = append(out, strings.Split(entryBlock, "\n")...)
	} else {
		replaceStart := commentBlockStart(lines, start)
		out = append(out, lines[:replaceStart]...)
		out = append(out, strings.Split(entryBlock, "\n")...)
		out = append(out, lines[end+1:]...)
	}

	result := strings.Join(out, "\n")
	if strings.Contains(config, "\r\n") {
		result = strings.ReplaceAll(result, "\n", "\r\n")
	}
	return result, nil
}
