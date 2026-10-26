package caddy

import (
	"strings"
	"testing"
)

// logSnippetTestConfig 模拟真实配置：全局块 → 其他 snippet 定义 → 站点块。
const logSnippetTestConfig = `{
  admin localhost:2019
  auto_https disable_redirects
}

(waf_protect) {
  coraza_waf {
    load_owasp_crs
  }
}

# 主站
app.example.com {
  import subdomain-log {
    name app
    size 10
    level INFO
  }
  reverse_proxy localhost:8080
}
`

const logSnippetTestOld = `(subdomain-log) {
  log {
    output file /var/log/caddy/{args.name}.log {
      roll_size {args.size}mb
      roll_keep 1
    }
    level {args.level}
    format json
  }
}`

// 无 snippet 时插到全局块之后、第一个站点块（含其上方注释）之前。
func TestMergeLogSnippetInsertsAfterGlobalBlock(t *testing.T) {
	config := "{\n  admin localhost:2019\n}\n\napp.example.com {\n  respond \"hi\"\n}\n"
	got, err := mergeCaddyLogSnippet(config, logSnippetTestOld)
	if err != nil {
		t.Fatal(err)
	}
	globalEnd := strings.Index(got, "admin localhost:2019")
	snippetAt := strings.Index(got, "(subdomain-log) {")
	siteAt := strings.Index(got, "app.example.com {")
	if globalEnd < 0 || snippetAt < 0 || siteAt < 0 {
		t.Fatalf("三段内容都应存在:\n%s", got)
	}
	if !(globalEnd < snippetAt && snippetAt < siteAt) {
		t.Fatalf("snippet 应位于全局块与站点块之间:\n%s", got)
	}
	if !strings.Contains(got, `respond "hi"`) {
		t.Fatalf("原站点块内容不应丢失:\n%s", got)
	}
}

// 已有定义整体替换，站点块里的 import 引用与其他内容不受影响。
func TestMergeLogSnippetReplacesExisting(t *testing.T) {
	config := "{\n  admin localhost:2019\n}\n\n(subdomain-log) {\n  log {\n    level DEBUG\n  }\n}\n\napp.example.com {\n  respond \"hi\"\n}\n"
	got, err := mergeCaddyLogSnippet(config, logSnippetTestOld)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got, "level DEBUG") {
		t.Fatalf("旧定义应被整体替换:\n%s", got)
	}
	if strings.Count(got, "(subdomain-log) {") != 1 {
		t.Fatalf("定义应恰好一份:\n%s", got)
	}
	if !strings.Contains(got, "roll_size {args.size}mb") || !strings.Contains(got, "roll_keep 1") {
		t.Fatalf("新定义内容应写入:\n%s", got)
	}
	if !strings.Contains(got, "app.example.com") || !strings.Contains(got, "admin localhost:2019") {
		t.Fatalf("其余内容不应受影响:\n%s", got)
	}
}

// 没有全局块时插到第一个站点块之前，且不把站点上方的注释卷进 snippet。
func TestMergeLogSnippetWithoutGlobalBlock(t *testing.T) {
	config := "# 主站\napp.example.com {\n  respond \"hi\"\n}\n"
	got, err := mergeCaddyLogSnippet(config, logSnippetTestOld)
	if err != nil {
		t.Fatal(err)
	}
	snippetAt := strings.Index(got, "(subdomain-log) {")
	commentAt := strings.Index(got, "# 主站")
	siteAt := strings.Index(got, "app.example.com {")
	if snippetAt < 0 {
		t.Fatalf("无全局块时应插入定义:\n%s", got)
	}
	if !(snippetAt < commentAt && commentAt < siteAt) {
		t.Fatalf("定义应插在注释与站点块之前，注释仍属于站点:\n%s", got)
	}
}

// 既没有全局块也没有站点块时追加到末尾。
func TestMergeLogSnippetAppendsWhenNoBlocks(t *testing.T) {
	got, err := mergeCaddyLogSnippet("# 只有注释\n", logSnippetTestOld)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(got, "# 只有注释") {
		t.Fatalf("原注释应保留在最前:\n%s", got)
	}
	if !strings.Contains(got, "(subdomain-log) {") {
		t.Fatalf("应追加定义:\n%s", got)
	}
}

func TestMergeLogSnippetIdempotent(t *testing.T) {
	once, err := mergeCaddyLogSnippet(logSnippetTestConfig, logSnippetTestOld)
	if err != nil {
		t.Fatal(err)
	}
	twice, err := mergeCaddyLogSnippet(once, logSnippetTestOld)
	if err != nil {
		t.Fatal(err)
	}
	if once != twice {
		t.Fatalf("合并应幂等\n--- once ---\n%s\n--- twice ---\n%s", once, twice)
	}
}

// CRLF 配置必须整篇保持 CRLF，不能混进孤立的 LF。
func TestMergeLogSnippetKeepsCRLF(t *testing.T) {
	config := strings.ReplaceAll("{\n  admin localhost:2019\n}\n\napp.example.com {\n  respond \"hi\"\n}\n", "\n", "\r\n")
	got, err := mergeCaddyLogSnippet(config, logSnippetTestOld)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "(subdomain-log) {\r\n") {
		t.Fatalf("插入内容应使用 CRLF:\n%q", got)
	}
	if strings.Count(got, "\n") != strings.Count(got, "\r\n") {
		t.Fatalf("不应出现孤立 LF:\n%q", got)
	}
}

// 空定义不改配置；非法定义必须报错而不是把坏字节写进配置。
func TestMergeLogSnippetValidation(t *testing.T) {
	got, err := mergeCaddyLogSnippet(logSnippetTestConfig, "  \n")
	if err != nil {
		t.Fatal(err)
	}
	if got != logSnippetTestConfig {
		t.Fatalf("空定义不应改动配置:\n%s", got)
	}
	if _, err := mergeCaddyLogSnippet(logSnippetTestConfig, "app.example.com {\n}\n"); err == nil {
		t.Fatal("块头不是 (subdomain-log) 应报错")
	}
	if _, err := mergeCaddyLogSnippet(logSnippetTestConfig, "(subdomain-log) {\n  log {\n"); err == nil {
		t.Fatal("花括号不配对应报错")
	}
}

// 名字前缀相同的其他 snippet 不能被当成目标块替换。
func TestMergeLogSnippetIgnoresLookalikeSnippet(t *testing.T) {
	lookalike := "(subdomain-logger) {\n  log {\n    level DEBUG\n  }\n}\n\n"
	got, err := mergeCaddyLogSnippet("{\n}\n\n"+lookalike+"app.example.com {\n  respond \"hi\"\n}\n", logSnippetTestOld)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "(subdomain-logger) {") || !strings.Contains(got, "level DEBUG") {
		t.Fatalf("同名前缀块不应被替换:\n%s", got)
	}
	if !strings.Contains(got, "(subdomain-log) {") {
		t.Fatalf("应另外插入目标定义:\n%s", got)
	}
}

// 默认定义必须带参数化骨架与 roll_keep 1，且块头能被识别。
func TestDefaultLogSnippetShape(t *testing.T) {
	snippet := defaultCaddyLogSnippetNormalized()
	for _, want := range []string{
		"(subdomain-log) {",
		"output file /var/log/caddy/{args.name}.log",
		"roll_size {args.size}mb",
		"roll_keep 1",
		"level {args.level}",
		"format json",
	} {
		if !strings.Contains(snippet, want) {
			t.Fatalf("默认定义应包含 %q:\n%s", want, snippet)
		}
	}
	if _, err := mergeCaddyLogSnippet("", snippet); err != nil {
		t.Fatalf("默认定义应能通过校验: %v", err)
	}
}
