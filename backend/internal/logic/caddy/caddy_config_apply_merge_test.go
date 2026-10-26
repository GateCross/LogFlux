package caddy

import (
	"strings"
	"testing"
)

// mergeCaddyManagedBlocks 是 Apply 在 adapt 前调用的合并入口，这里覆盖
// 全局选项、入口块与日志 snippet 在同一份配置上叠加的最终形态。
func TestMergeManagedBlocksCombinesOptionsAndEntry(t *testing.T) {
	debugOn := true
	opts := CaddyGlobalOptionsValue{
		AutoHttps: "disable_redirects",
		Debug:     &debugOn,
	}
	entryBlock := ":80 {\n  respond \"entry\"\n}"
	got, err := mergeCaddyManagedBlocks(mergeTestBaseConfig, opts, entryBlock, "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "auto_https disable_redirects") || !strings.Contains(got, "  debug") {
		t.Fatalf("全局选项应合并进全局块:\n%s", got)
	}
	if !strings.Contains(got, `respond "entry"`) {
		t.Fatalf("入口块应替换为模板:\n%s", got)
	}
	if strings.Contains(got, "import waf_protect") {
		t.Fatalf("旧 :80 块内容应被替换:\n%s", got)
	}
}

func TestMergeManagedBlocksAddsLogSnippet(t *testing.T) {
	snippet := "(subdomain-log) {\n  log {\n    level INFO\n  }\n}"
	got, err := mergeCaddyManagedBlocks(mergeTestBaseConfig, CaddyGlobalOptionsValue{}, "", snippet)
	if err != nil {
		t.Fatal(err)
	}
	// 全局块在最前，snippet 紧随其后，站点块最后。
	if strings.Index(got, "admin localhost:2019") > strings.Index(got, "(subdomain-log)") {
		t.Fatalf("snippet 应在全局块之后:\n%s", got)
	}
	if strings.Index(got, "(subdomain-log)") > strings.Index(got, ":80 {") {
		t.Fatalf("snippet 应在第一个站点块之前:\n%s", got)
	}
}

func TestMergeManagedBlocksEmptyKeepsConfig(t *testing.T) {
	got, err := mergeCaddyManagedBlocks(mergeTestBaseConfig, CaddyGlobalOptionsValue{}, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if got != mergeTestBaseConfig {
		t.Fatalf("空选项与空模板不应改动配置:\n%s", got)
	}
}

// 合并入口在 snippet 就位后顺带把站点侧旧 import 写法迁移为三参数单行。
func TestMergeManagedBlocksMigratesOldLogImports(t *testing.T) {
	config := ":80 {\n  import subdomain-log local\n}\n\napp.example.com {\n  import subdomain-log {\n    name app\n    size 10\n    level INFO\n  }\n}\n"
	got, err := mergeCaddyManagedBlocks(config, CaddyGlobalOptionsValue{}, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "import subdomain-log local 10 DEBUG") {
		t.Fatalf("单行 1 参数 import 应被迁移:\n%s", got)
	}
	if !strings.Contains(got, "import subdomain-log app 10 INFO") {
		t.Fatalf("块形式 import 应被迁移:\n%s", got)
	}
	if strings.Contains(got, "name app") {
		t.Fatalf("块形式不应残留:\n%s", got)
	}
}

func TestMergeManagedBlocksRejectsBadEntryTemplate(t *testing.T) {
	if _, err := mergeCaddyManagedBlocks(mergeTestBaseConfig, CaddyGlobalOptionsValue{}, "example.com {\n}\n", ""); err == nil {
		t.Fatal("缺 :80 地址的模板应报错")
	}
}

func TestExtractCaddyEntryBlock(t *testing.T) {
	got, err := extractCaddyEntryBlock(entryTestConfig)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(got, ":80 {") || !strings.HasSuffix(got, "}") {
		t.Fatalf("提取结果应是完整 :80 块:\n%s", got)
	}
	if strings.Contains(got, "admin localhost:2019") || strings.Contains(got, "pve.example.com") {
		t.Fatalf("提取不应包含块外内容:\n%s", got)
	}
	if _, err := extractCaddyEntryBlock("example.com {\n}\n"); err == nil {
		t.Fatal("无 :80 块应报错")
	}
}
