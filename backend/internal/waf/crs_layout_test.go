package waf

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// buildTestCRSRelease 在临时目录里构造一份官方 CRS 发布包结构：
//
//	root/coreruleset-4.25.0/rules/REQUEST-901-INITIALIZATION.conf
//	root/coreruleset-4.25.0/rules/php-variables.data
//	root/coreruleset-4.25.0/crs-setup.conf.example
func buildTestCRSRelease(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	rulesDir := filepath.Join(root, "coreruleset-4.25.0", "rules")
	if err := os.MkdirAll(rulesDir, 0o755); err != nil {
		t.Fatalf("创建规则目录失败: %v", err)
	}

	files := map[string]string{
		"REQUEST-901-INITIALIZATION.conf":                     "SecRule REQUEST_URI \"@rx ^/$\" \"id:1001,phase:1,pass\"\n",
		"REQUEST-942-APPLICATION-ATTACK-SQLI.conf":            "@pmFromFile sql-errors.data\n",
		"php-variables.data":                                  "$_GET\n",
		"sql-errors.data":                                     "error\n",
		"REQUEST-900-EXCLUSION-RULES-BEFORE-CRS.conf.example": "# 模板，不应被加载\n",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(rulesDir, name), []byte(content), 0o644); err != nil {
			t.Fatalf("写入 %s 失败: %v", name, err)
		}
	}

	setupPath := filepath.Join(root, "coreruleset-4.25.0", "crs-setup.conf.example")
	if err := os.WriteFile(setupPath, []byte("SecAction \"id:900000,pass\"\n"), 0o644); err != nil {
		t.Fatalf("写入 setup 失败: %v", err)
	}

	return root
}

func TestBuildCRSLayout(t *testing.T) {
	releaseDir := buildTestCRSRelease(t)
	layoutRoot := filepath.Join(t.TempDir(), "active")

	layout, err := BuildCRSLayout(releaseDir, layoutRoot)
	if err != nil {
		t.Fatalf("BuildCRSLayout() error = %v", err)
	}

	if layout.Root != filepath.Clean(layoutRoot) {
		t.Fatalf("unexpected root: %s", layout.Root)
	}

	// .conf 与 .data 必须位于同一目录，否则规则内的 @pmFromFile 找不到数据文件。
	if layout.OWASPDir != filepath.Join(layoutRoot, "rules", "@owasp_crs") {
		t.Fatalf("unexpected owasp dir: %s", layout.OWASPDir)
	}
	if !layout.HasRules() {
		t.Fatalf("expected rules in layout")
	}

	// .conf.example 是模板，不参与加载。
	for _, file := range layout.RuleFiles {
		if strings.HasSuffix(file, ".example") {
			t.Fatalf("example file should not be treated as rule: %s", file)
		}
	}
	if len(layout.RuleFiles) != 2 {
		t.Fatalf("expected 2 rule files, got %d (%v)", len(layout.RuleFiles), layout.RuleFiles)
	}
	if len(layout.DataFiles) != 2 {
		t.Fatalf("expected 2 data files, got %d (%v)", len(layout.DataFiles), layout.DataFiles)
	}

	for _, file := range append(append([]string{}, layout.RuleFiles...), layout.DataFiles...) {
		if _, err := os.Stat(file); err != nil {
			t.Fatalf("layout file missing: %s err=%v", file, err)
		}
	}

	if got := layout.SetupConfigPath(); got != filepath.Join(layoutRoot, "rules", "@crs-setup.conf.example") {
		t.Fatalf("unexpected setup path: %s", got)
	}
	if got := layout.CorazaConfigPath(); got != "" {
		t.Fatalf("expected empty coraza config path, got %s", got)
	}
}

func TestBuildCRSLayoutMissingRules(t *testing.T) {
	emptyDir := t.TempDir()
	if _, err := BuildCRSLayout(emptyDir, filepath.Join(t.TempDir(), "active")); err == nil {
		t.Fatalf("expected error for missing rules")
	}
}

func TestBuildCRSLayoutAcceptsRulesDir(t *testing.T) {
	releaseDir := buildTestCRSRelease(t)
	rulesDir := filepath.Join(releaseDir, "coreruleset-4.25.0", "rules")

	layout, err := BuildCRSLayout(rulesDir, filepath.Join(t.TempDir(), "active"))
	if err != nil {
		t.Fatalf("BuildCRSLayout() error = %v", err)
	}
	if !layout.HasRules() {
		t.Fatalf("expected rules when source is rules dir")
	}
}

func TestActivatorBuildsLayoutBeforeLoad(t *testing.T) {
	store := NewStore(t.TempDir())
	releaseDir := store.ReleaseDir("v4.25.0")
	if err := os.MkdirAll(filepath.Join(releaseDir, "coreruleset-4.25.0", "rules"), 0o755); err != nil {
		t.Fatalf("创建发布目录失败: %v", err)
	}
	if err := os.WriteFile(filepath.Join(releaseDir, "coreruleset-4.25.0", "rules", "REQUEST-901-INITIALIZATION.conf"), []byte("SecAction \"id:1,pass\"\n"), 0o644); err != nil {
		t.Fatalf("写入规则失败: %v", err)
	}

	var adaptCalls, loadCalls int
	loader := &recordingCaddyLoader{
		onAdapt: func(config string) error {
			adaptCalls++
			// 布局必须在 Adapt 之前就绪，否则 Caddyfile 的 Include 会指到不存在的路径。
			if _, err := os.Stat(filepath.Join(store.BaseDir, "active", "rules", "@owasp_crs", "REQUEST-901-INITIALIZATION.conf")); err != nil {
				t.Errorf("layout not ready before adapt: %v", err)
			}
			return nil
		},
		onLoad: func(config string) error {
			loadCalls++
			return nil
		},
	}

	activator := &Activator{Store: store, CaddyLoader: loader}
	if err := activator.ActivateVersion("v4.25.0", "{}"); err != nil {
		t.Fatalf("ActivateVersion() error = %v", err)
	}
	if adaptCalls != 1 || loadCalls != 1 {
		t.Fatalf("unexpected adapt/load counts: %d/%d", adaptCalls, loadCalls)
	}

	currentTarget, err := store.LinkTarget(store.CurrentLinkPath())
	if err != nil {
		t.Fatalf("LinkTarget() error = %v", err)
	}
	if currentTarget != filepath.Clean(releaseDir) {
		t.Fatalf("unexpected current target: %s", currentTarget)
	}
}

func TestActivatorLayoutRootOutsideRelease(t *testing.T) {
	store := NewStore(t.TempDir())
	activator := &Activator{Store: store}

	layoutRoot := activator.LayoutRoot()
	if layoutRoot != filepath.Join(store.BaseDir, "active") {
		t.Fatalf("unexpected layout root: %s", layoutRoot)
	}
	// 布局不能落在 release 目录内，否则会污染版本内容并阻碍回滚。
	if strings.HasPrefix(layoutRoot, store.ReleasesDir) {
		t.Fatalf("layout root must not be inside releases dir: %s", layoutRoot)
	}
}

type recordingCaddyLoader struct {
	onAdapt func(config string) error
	onLoad  func(config string) error
}

func (l *recordingCaddyLoader) Adapt(config string) error {
	if l.onAdapt != nil {
		return l.onAdapt(config)
	}
	return nil
}

func (l *recordingCaddyLoader) Load(config string) error {
	if l.onLoad != nil {
		return l.onLoad(config)
	}
	return nil
}
