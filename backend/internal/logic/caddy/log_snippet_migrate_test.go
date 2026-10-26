package caddy

import (
	"strings"
	"testing"
)

// 单行 1 参数补齐 10 DEBUG，缩进保留。
func TestMigrateSiteLogImportsSingleArg(t *testing.T) {
	config := "app.example.com {\n  import subdomain-log app\n  reverse_proxy localhost:8080\n}\n"
	got := migrateSiteLogImports(config)
	want := "app.example.com {\n  import subdomain-log app 10 DEBUG\n  reverse_proxy localhost:8080\n}\n"
	if got != want {
		t.Fatalf("单行 1 参数应补齐 10 DEBUG:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

// 块形式整块替换为单行，键值原样转为位置参数。
func TestMigrateSiteLogImportsBlockForm(t *testing.T) {
	config := "app.example.com {\n  import subdomain-log {\n    name app\n    size 10\n    level INFO\n  }\n  reverse_proxy localhost:8080\n}\n"
	got := migrateSiteLogImports(config)
	want := "app.example.com {\n  import subdomain-log app 10 INFO\n  reverse_proxy localhost:8080\n}\n"
	if got != want {
		t.Fatalf("块形式应替换为单行:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

// 块形式缺 size / level 时按旧模板现状补齐。
func TestMigrateSiteLogImportsBlockFormDefaults(t *testing.T) {
	config := "app.example.com {\n  import subdomain-log {\n    name app\n  }\n}\n"
	got := migrateSiteLogImports(config)
	if !strings.Contains(got, "import subdomain-log app 10 DEBUG") {
		t.Fatalf("缺 size/level 应补 10 DEBUG:\n%s", got)
	}
	if strings.Contains(got, "name app") {
		t.Fatalf("键值行不应残留:\n%s", got)
	}
}

// 块形式缺 name 时无法构造合法 import，整块保持原样。
func TestMigrateSiteLogImportsBlockWithoutNameUnchanged(t *testing.T) {
	config := "app.example.com {\n  import subdomain-log {\n    size 10\n    level INFO\n  }\n}\n"
	if got := migrateSiteLogImports(config); got != config {
		t.Fatalf("缺 name 的块应保持原样:\n--- got ---\n%s\n--- want ---\n%s", got, config)
	}
}

// 已经是 3 参数的单行不动；2 参数等不明形态也不动。
func TestMigrateSiteLogImportsUntouchedForms(t *testing.T) {
	migrated := "app.example.com {\n  import subdomain-log app 10 INFO\n}\n"
	if got := migrateSiteLogImports(migrated); got != migrated {
		t.Fatalf("3 参数单行不应改动:\n--- got ---\n%s\n--- want ---\n%s", got, migrated)
	}
	twoArgs := "app.example.com {\n  import subdomain-log app 10\n}\n"
	if got := migrateSiteLogImports(twoArgs); got != twoArgs {
		t.Fatalf("2 参数形态不明，不应改动:\n--- got ---\n%s\n--- want ---\n%s", got, twoArgs)
	}
	bare := "app.example.com {\n  import subdomain-log\n}\n"
	if got := migrateSiteLogImports(bare); got != bare {
		t.Fatalf("无参数 import 无法补 name，不应改动:\n--- got ---\n%s\n--- want ---\n%s", got, bare)
	}
}

// 行缩进与 CRLF 风格原样保留。
func TestMigrateSiteLogImportsKeepsIndentAndCRLF(t *testing.T) {
	config := strings.ReplaceAll("app.example.com {\n\timport subdomain-log app\n}\n", "\n", "\r\n")
	got := migrateSiteLogImports(config)
	want := strings.ReplaceAll("app.example.com {\n\timport subdomain-log app 10 DEBUG\n}\n", "\n", "\r\n")
	if got != want {
		t.Fatalf("缩进与 CRLF 应保留:\n--- got ---\n%q\n--- want ---\n%q", got, want)
	}
	if strings.Count(got, "\n") != strings.Count(got, "\r\n") {
		t.Fatalf("不应出现孤立 LF:\n%q", got)
	}
}

// 幂等：迁移一次后再跑不产生新变化。
func TestMigrateSiteLogImportsIdempotent(t *testing.T) {
	config := "app.example.com {\n  import subdomain-log app\n}\nblog.example.com {\n  import subdomain-log {\n    name blog\n    size 20\n    level ERROR\n  }\n}\n"
	once := migrateSiteLogImports(config)
	twice := migrateSiteLogImports(once)
	if once != twice {
		t.Fatalf("迁移应幂等\n--- once ---\n%s\n--- twice ---\n%s", once, twice)
	}
}

// 多站点各自迁移，互不影响。
func TestMigrateSiteLogImportsMultiSite(t *testing.T) {
	config := "app.example.com {\n  import subdomain-log app\n}\n\nblog.example.com {\n  import subdomain-log {\n    name blog\n    size 20\n    level ERROR\n  }\n}\n"
	got := migrateSiteLogImports(config)
	if !strings.Contains(got, "import subdomain-log app 10 DEBUG") {
		t.Fatalf("站点 1 应迁移:\n%s", got)
	}
	if !strings.Contains(got, "import subdomain-log blog 20 ERROR") {
		t.Fatalf("站点 2 应迁移:\n%s", got)
	}
	if strings.Contains(got, "name blog") {
		t.Fatalf("块形式不应残留:\n%s", got)
	}
}

// 同名前缀的其他 import 与 snippet 定义不受影响。
func TestMigrateSiteLogImportsIgnoresLookalikes(t *testing.T) {
	config := "(subdomain-logger) {\n  log {\n    level DEBUG\n  }\n}\n\napp.example.com {\n  import subdomain-logger app\n  import subdomain-log app\n}\n"
	got := migrateSiteLogImports(config)
	if !strings.Contains(got, "import subdomain-logger app") {
		t.Fatalf("同名前缀 import 不应改动:\n%s", got)
	}
	if !strings.Contains(got, "import subdomain-log app 10 DEBUG") {
		t.Fatalf("目标 import 应迁移:\n%s", got)
	}
}
