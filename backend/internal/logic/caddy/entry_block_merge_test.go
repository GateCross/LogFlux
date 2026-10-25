package caddy

import (
	"strings"
	"testing"
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
	if !strings.Contains(block, "import frontend_simple") || !strings.Contains(block, "@lan") {
		t.Fatalf("默认模板应含本体服务与内网分流骨架:\n%s", block)
	}
}
