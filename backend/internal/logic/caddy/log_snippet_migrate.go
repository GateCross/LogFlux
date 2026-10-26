package caddy

import "strings"

// 存量 import 迁移时缺省参数的补齐值：旧站点模板 level 固定 debug，
// 补 DEBUG 保持日志量不骤减；size 沿用旧模板的固定 10MB。
const (
	migratedImportDefaultSize  = "10"
	migratedImportDefaultLevel = "DEBUG"
)

// migrateSiteLogImports 把站点侧旧 import 写法整体迁移为三参数单行形式。
//
// 处理规则：
//   - 单行 1 参数 import subdomain-log <name> → 补 10 DEBUG
//   - 多行块 import subdomain-log { name X size Y level Z } → 整块替换为单行 import subdomain-log X Y Z
//   - 已经是 3 参数的单行与其它形态不动
//
// 保留行缩进与 CRLF；幂等，可对同一配置重复执行。
func migrateSiteLogImports(config string) string {
	if !strings.Contains(config, caddyLogSnippetName) {
		return config
	}
	normalized := strings.ReplaceAll(config, "\r\n", "\n")
	lines := strings.Split(normalized, "\n")
	out := make([]string, 0, len(lines))

	for idx := 0; idx < len(lines); idx++ {
		line := lines[idx]
		args, comment, blockOpen, isImport := parseSiteLogImportLine(line)
		if !isImport {
			out = append(out, line)
			continue
		}

		// 块形式：定位收尾 } 并整块替换为单行；块未闭合或缺 name 时保持原样
		if blockOpen {
			end, values, ok := collectSiteLogImportBlock(lines, idx)
			if !ok || values["name"] == "" {
				out = append(out, line)
				continue
			}
			args = []string{
				values["name"],
				orDefaultString(values["size"], migratedImportDefaultSize),
				orDefaultString(values["level"], migratedImportDefaultLevel),
			}
			out = append(out, renderSiteLogImportLine(line, args, comment))
			idx = end
			continue
		}

		switch len(args) {
		case 1:
			// 旧单行写法只有站点名，size/level 按旧模板现状补齐
			args = append(args, migratedImportDefaultSize, migratedImportDefaultLevel)
			out = append(out, renderSiteLogImportLine(line, args, comment))
		default:
			// 已迁移（3 参数）或形态不明（空参/2 参/超参），不动
			out = append(out, line)
		}
	}

	result := strings.Join(out, "\n")
	if strings.Contains(config, "\r\n") {
		result = strings.ReplaceAll(result, "\n", "\r\n")
	}
	return result
}

// parseSiteLogImportLine 判断一行是否为站点日志 import。
// 返回参数 token（去掉 import 与 snippet 名）、行尾注释、是否块形式开头。
func parseSiteLogImportLine(line string) (args []string, comment string, blockOpen bool, ok bool) {
	fields := strings.Fields(strings.TrimSpace(line))
	if len(fields) < 2 || fields[0] != "import" || fields[1] != caddyLogSnippetName {
		return nil, "", false, false
	}
	rest := fields[2:]
	// 行尾注释从第一个 # token 起剥离
	for i, f := range rest {
		if strings.HasPrefix(f, "#") {
			comment = strings.Join(rest[i:], " ")
			rest = rest[:i]
			break
		}
	}
	if len(rest) > 0 && rest[0] == "{" {
		return nil, comment, true, true
	}
	return rest, comment, false, true
}

// collectSiteLogImportBlock 从块头下一行开始收集 name/size/level，
// 返回收尾 } 所在行下标；块未闭合返回 ok=false。
func collectSiteLogImportBlock(lines []string, openIdx int) (int, map[string]string, bool) {
	values := map[string]string{}
	depth := 1
	for idx := openIdx + 1; idx < len(lines); idx++ {
		trimmed := strings.TrimSpace(lines[idx])
		depth += strings.Count(trimmed, "{") - strings.Count(trimmed, "}")
		if depth <= 0 {
			return idx, values, true
		}
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.ContainsAny(trimmed, "{}") {
			continue
		}
		fields := strings.Fields(trimmed)
		if len(fields) != 2 {
			continue
		}
		switch strings.ToLower(fields[0]) {
		case "name", "size", "level":
			values[strings.ToLower(fields[0])] = fields[1]
		}
	}
	return -1, nil, false
}

// renderSiteLogImportLine 按三参数渲染迁移后的单行，保留原缩进与行尾注释。
func renderSiteLogImportLine(line string, args []string, comment string) string {
	indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
	migrated := indent + "import " + caddyLogSnippetName + " " + strings.Join(args, " ")
	if comment != "" {
		migrated += " " + comment
	}
	return migrated
}

func orDefaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
