package waf

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

type mockCaddyLoader struct {
	adaptCalls int
	loadCalls  int
	adaptErr   error
	loadErr    error
}

func (loader *mockCaddyLoader) Adapt(config string) error {
	loader.adaptCalls++
	return loader.adaptErr
}

func (loader *mockCaddyLoader) Load(config string) error {
	loader.loadCalls++
	return loader.loadErr
}

func TestActivatorActivateVersionSuccess(t *testing.T) {
	store, oldReleaseDir, newReleaseDir := prepareActivatorStore(t)
	loader := &mockCaddyLoader{}
	activator := &Activator{Store: store, CaddyLoader: loader}

	if err := activator.ActivateVersion("v4.23.1", "example config"); err != nil {
		t.Fatalf("ActivateVersion returned error: %v", err)
	}

	currentTarget, err := store.LinkTarget(store.CurrentLinkPath())
	if err != nil {
		t.Fatalf("resolve current link failed: %v", err)
	}
	if currentTarget != filepath.Clean(newReleaseDir) {
		t.Fatalf("unexpected current target: %s", currentTarget)
	}

	lastGoodTarget, err := store.LinkTarget(store.LastGoodLinkPath())
	if err != nil {
		t.Fatalf("resolve last_good link failed: %v", err)
	}
	if lastGoodTarget != filepath.Clean(oldReleaseDir) {
		t.Fatalf("unexpected last_good target: %s", lastGoodTarget)
	}

	if loader.adaptCalls != 1 || loader.loadCalls != 1 {
		t.Fatalf("unexpected calls: adapt=%d load=%d", loader.adaptCalls, loader.loadCalls)
	}
}

func TestActivatorActivateVersionRollbackWhenAdaptFailed(t *testing.T) {
	store, oldReleaseDir, _ := prepareActivatorStore(t)
	loader := &mockCaddyLoader{adaptErr: fmt.Errorf("adapt failed")}
	activator := &Activator{Store: store, CaddyLoader: loader}

	err := activator.ActivateVersion("v4.23.1", "example config")
	if err == nil {
		t.Fatalf("expected activate error")
	}

	currentTarget, err := store.LinkTarget(store.CurrentLinkPath())
	if err != nil {
		t.Fatalf("resolve current link failed: %v", err)
	}
	if currentTarget != filepath.Clean(oldReleaseDir) {
		t.Fatalf("expected rollback to old release, got %s", currentTarget)
	}

	if loader.adaptCalls != 1 || loader.loadCalls != 1 {
		t.Fatalf("unexpected calls: adapt=%d load=%d", loader.adaptCalls, loader.loadCalls)
	}
}

func TestActivatorActivateVersionRollbackWhenLoadFailed(t *testing.T) {
	store, oldReleaseDir, _ := prepareActivatorStore(t)
	loader := &mockCaddyLoader{loadErr: fmt.Errorf("load failed")}
	activator := &Activator{Store: store, CaddyLoader: loader}

	err := activator.ActivateVersion("v4.23.1", "example config")
	if err == nil {
		t.Fatalf("expected activate error")
	}

	currentTarget, err := store.LinkTarget(store.CurrentLinkPath())
	if err != nil {
		t.Fatalf("resolve current link failed: %v", err)
	}
	if currentTarget != filepath.Clean(oldReleaseDir) {
		t.Fatalf("expected rollback to old release, got %s", currentTarget)
	}

	if loader.adaptCalls != 1 || loader.loadCalls != 2 {
		t.Fatalf("unexpected calls: adapt=%d load=%d", loader.adaptCalls, loader.loadCalls)
	}
}

func prepareActivatorStore(t *testing.T) (*Store, string, string) {
	t.Helper()
	store := NewStore(filepath.Join(t.TempDir(), "waf"))
	if err := store.EnsureDirs(); err != nil {
		t.Fatalf("EnsureDirs returned error: %v", err)
	}

	oldReleaseDir := store.ReleaseDir("v4.23.0")
	newReleaseDir := store.ReleaseDir("v4.23.1")
	writeTestReleaseRules(t, oldReleaseDir)
	writeTestReleaseRules(t, newReleaseDir)

	if err := store.SetLink(store.CurrentLinkPath(), oldReleaseDir); err != nil {
		t.Fatalf("set initial current link failed: %v", err)
	}

	return store, oldReleaseDir, newReleaseDir
}

// writeTestReleaseRules 在发布目录里写一份最小可用的 CRS 规则树。
//
// 激活流程会先把发布目录重构成布局再加载配置，因此发布目录必须含规则文件，
// 否则激活会在构建布局阶段直接失败。
func writeTestReleaseRules(t *testing.T, releaseDir string) {
	t.Helper()

	rulesDir := filepath.Join(releaseDir, "coreruleset-4.23.1", "rules")
	if err := os.MkdirAll(rulesDir, 0o755); err != nil {
		t.Fatalf("create rules dir failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rulesDir, "REQUEST-901-INITIALIZATION.conf"), []byte("SecAction \"id:1,pass\"\n"), 0o644); err != nil {
		t.Fatalf("write rule failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rulesDir, "crs-setup.conf.example"), []byte("SecAction \"id:2,pass\"\n"), 0o644); err != nil {
		t.Fatalf("write setup failed: %v", err)
	}
}
