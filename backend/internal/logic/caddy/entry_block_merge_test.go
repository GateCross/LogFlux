package caddy

import (
	"regexp"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"logflux/internal/svc"
)

const entryTestConfig = `{
  admin localhost:2019
}

:80 {
  import waf_protect
  respond "old entry"
}

pve.example.com {
  import cloudflare
}
`

func TestMergeEntryBlockReplacesExisting(t *testing.T) {
	newBlock := ":80 {\n  respond \"new\"\n}"
	got, err := mergeEntryBlock(entryTestConfig, newBlock)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got, "old entry") {
		t.Fatalf("旧块应被替换:\n%s", got)
	}
	if !strings.Contains(got, `respond "new"`) {
		t.Fatalf("新块应存在:\n%s", got)
	}
	if !strings.Contains(got, "pve.example.com") || !strings.Contains(got, "admin localhost:2019") {
		t.Fatalf("其余内容不应受影响:\n%s", got)
	}
}

func TestMergeEntryBlockIdempotent(t *testing.T) {
	block := ":80 {\n  respond \"x\"\n}"
	once, err := mergeEntryBlock(entryTestConfig, block)
	if err != nil {
		t.Fatal(err)
	}
	twice, err := mergeEntryBlock(once, block)
	if err != nil {
		t.Fatal(err)
	}
	if once != twice {
		t.Fatalf("合并应幂等\n%s\n---\n%s", once, twice)
	}
}

func TestMergeEntryBlockAppendsWhenMissing(t *testing.T) {
	config := "example.com {\n  respond \"hi\"\n}\n"
	got, err := mergeEntryBlock(config, ":80 {\n  respond \"entry\"\n}")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "example.com") || !strings.Contains(got, `respond "entry"`) {
		t.Fatalf("应追加而不丢失原内容:\n%s", got)
	}
}

func TestMergeEntryBlockRejectsWrongTemplate(t *testing.T) {
	if _, err := mergeEntryBlock(entryTestConfig, "example.com {\n}\n"); err == nil {
		t.Fatal("缺少 :80 地址的模板应报错")
	}
}

func TestMergeEntryBlockEmptyTemplateKeepsConfig(t *testing.T) {
	got, err := mergeEntryBlock(entryTestConfig, "  \n")
	if err != nil {
		t.Fatal(err)
	}
	if got != entryTestConfig {
		t.Fatalf("空模板不应改动配置:\n%s", got)
	}
}

func TestMergeEntryBlockDefaultTemplateEndsWithEntry(t *testing.T) {
	block := defaultCaddyEntryBlockNormalized()
	if !strings.HasPrefix(block, "# :80") && !strings.HasPrefix(block, ":80") {
		t.Fatalf("默认模板开头异常:\n%s", block[:40])
	}
	// 本体服务已内联，不再 import 未定义的业务 snippet。
	if !strings.Contains(block, "file_server") || !strings.Contains(block, "@lan") {
		t.Fatalf("默认模板应含本体服务与内网分流骨架:\n%s", block)
	}
	for _, dangling := range []string{"import frontend_simple", "import api_proxy", "import health_check", "import compression"} {
		if strings.Contains(block, dangling) {
			t.Fatalf("默认模板不应再有悬空 import %q:\n%s", dangling, block)
		}
	}
}

// systemConfigsEmptyRows 造一份空结果集，供「配置键不存在」的默认分支使用。
func systemConfigsEmptyRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{"id", "created_at", "updated_at", "key", "value"})
}

// 悬空 import 属性测试：合并结果里每个 import 的名字都必须有同名 snippet 定义。
//
// 这是能拦住同类回归的最小检查——以后谁往入口块模板里加业务 snippet 而忘了定义，
// 这条测试直接红，而不是等用户点保存时拿到一句晦涩的 snippet not found。
//
// 走 mergeManagedBlocks 而非纯 mergeCaddyManagedBlocks：waf_protect 由前者补，
// 纯函数版本只做合并，拿不到这层保证。
func TestMergeManagedBlocksNoDanglingImports(t *testing.T) {
	db, mock, cleanup := newCaddySqlmockDB(t)
	defer cleanup()

	// 入口块模板、全局选项、ip_region 开关、日志 snippet 各读一次 system_configs，
	// 全部返回空结果集走内置默认值。
	for i := 0; i < 4; i++ {
		mock.ExpectQuery(`SELECT \* FROM "system_configs"`).WillReturnRows(systemConfigsEmptyRows())
	}

	service := newCaddyConfigApplyService(&svc.ServiceContext{DB: db}, nil)
	merged, err := service.mergeManagedBlocks("")
	if err != nil {
		t.Fatalf("mergeManagedBlocks() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql 期望未满足: %v", err)
	}

	importPattern := regexp.MustCompile(`(?m)^\s*import\s+(\S+)`)
	matches := importPattern.FindAllStringSubmatch(merged, -1)
	if len(matches) == 0 {
		t.Fatalf("默认模板应至少有一个 import:\n%s", merged)
	}
	for _, match := range matches {
		name := match[1]
		if !strings.Contains(merged, "("+name+") {") {
			t.Fatalf("import %q 没有对应的 snippet 定义:\n%s", name, merged)
		}
	}
}

// ensureSnippetDefinition 只看块名：用户已定义的同名 snippet 必须字节级不变。
//
// 故意用 include 文件写法（不含 directives 反引号）——旧 ensureWafProtectSnippet
// 正是按内容判定，会把这种写法整块替换成 DetectionOnly，WAF 从拦截变只检测。
func TestEnsureSnippetDefinitionKeepsUserContent(t *testing.T) {
	userSnippet := "(waf_protect) {\n  coraza_waf {\n    load_owasp_crs\n    include /x.conf\n  }\n}\n"
	config := userSnippet + "\n:80 {\n  import waf_protect\n}\n"

	got, changed, err := ensureSnippetDefinition(config, "waf_protect", renderWafProtectSnippet("\n"))
	if err != nil {
		t.Fatalf("ensureSnippetDefinition() error = %v", err)
	}
	if changed {
		t.Fatalf("已有定义时不应报告改动")
	}
	if got != config {
		t.Fatalf("用户内容必须原样保留:\nwant:\n%s\ngot:\n%s", config, got)
	}
}

// removeCaddyDirectiveLine 只删指定指令行，其余行原样保留。
func TestRemoveCaddyDirectiveLine(t *testing.T) {
	config := defaultCaddyEntryBlockNormalized()
	if !strings.Contains(config, "import geo_auth") {
		t.Fatalf("默认模板应含 geo_auth import:\n%s", config)
	}

	got := removeCaddyDirectiveLine(config, "import geo_auth")

	for _, line := range strings.Split(got, "\n") {
		if strings.TrimSpace(line) == "import geo_auth" {
			t.Fatalf("geo_auth import 应被删除:\n%s", got)
		}
	}
	for _, keep := range []string{"import waf_protect", "import subdomain-log local 10 INFO", "file_server"} {
		if !strings.Contains(got, keep) {
			t.Fatalf("其余行不应受影响，缺少 %q:\n%s", keep, got)
		}
	}
}

// geo_auth 跟随 ip_region 开关：关闭时既无 import 也无定义，开启时两者都在。
func TestMergeManagedBlocksGeoAuthFollowsIPRegionSwitch(t *testing.T) {
	t.Run("ip_region 关闭", func(t *testing.T) {
		db, mock, cleanup := newCaddySqlmockDB(t)
		defer cleanup()
		mock.ExpectQuery(`SELECT \* FROM "system_configs"`).WillReturnRows(systemConfigsEmptyRows())
		mock.ExpectQuery(`SELECT \* FROM "system_configs"`).WillReturnRows(systemConfigsEmptyRows())
		mock.ExpectQuery(`SELECT \* FROM "system_configs"`).WillReturnRows(systemConfigsEmptyRows())
		mock.ExpectQuery(`SELECT \* FROM "system_configs"`).WillReturnRows(systemConfigsEmptyRows())

		service := newCaddyConfigApplyService(&svc.ServiceContext{DB: db}, nil)
		merged, err := service.mergeManagedBlocks("")
		if err != nil {
			t.Fatalf("mergeManagedBlocks() error = %v", err)
		}
		if strings.Contains(merged, "import geo_auth") {
			t.Fatalf("开关关闭时不应保留 geo_auth import:\n%s", merged)
		}
		if strings.Contains(merged, "(geo_auth) {") {
			t.Fatalf("开关关闭时不应补 geo_auth 定义:\n%s", merged)
		}
	})

	t.Run("ip_region 开启", func(t *testing.T) {
		db, mock, cleanup := newCaddySqlmockDB(t)
		defer cleanup()
		// 入口块模板、全局选项、日志 snippet 走默认值，只有 ip_region 有值。
		ipRegionRows := func() *sqlmock.Rows {
			return sqlmock.NewRows([]string{"id", "created_at", "updated_at", "key", "value"}).
				AddRow(uint(1), nil, nil, "ip_region", `{"enabled":true}`)
		}
		mock.MatchExpectationsInOrder(false)
		mock.ExpectQuery(`SELECT \* FROM "system_configs" WHERE key = \$1`).
			WithArgs("ip_region", 1).
			WillReturnRows(ipRegionRows())
		for i := 0; i < 3; i++ {
			mock.ExpectQuery(`SELECT \* FROM "system_configs" WHERE key = \$1`).
				WillReturnRows(systemConfigsEmptyRows())
		}

		service := newCaddyConfigApplyService(&svc.ServiceContext{DB: db}, nil)
		merged, err := service.mergeManagedBlocks("")
		if err != nil {
			t.Fatalf("mergeManagedBlocks() error = %v", err)
		}
		if !strings.Contains(merged, "import geo_auth") {
			t.Fatalf("开关开启时应保留 geo_auth import:\n%s", merged)
		}
		if !strings.Contains(merged, "(geo_auth) {") {
			t.Fatalf("开关开启时应补 geo_auth 定义:\n%s", merged)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("sql 期望未满足: %v", err)
		}
	})
}
