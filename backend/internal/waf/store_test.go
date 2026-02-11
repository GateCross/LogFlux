package waf

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStoreEnsureDirs(t *testing.T) {
	baseDir := filepath.Join(t.TempDir(), "waf")
	store := NewStore(baseDir)

	if err := store.EnsureDirs(); err != nil {
		t.Fatalf("EnsureDirs returned error: %v", err)
	}

	directories := []string{store.BaseDir, store.PackagesDir, store.ReleasesDir}
	for _, directory := range directories {
		info, err := os.Stat(directory)
		if err != nil {
			t.Fatalf("stat dir failed: %s, err=%v", directory, err)
		}
		if !info.IsDir() {
			t.Fatalf("expected directory: %s", directory)
		}
	}
}

func TestStoreLink(t *testing.T) {
	baseDir := filepath.Join(t.TempDir(), "waf")
	store := NewStore(baseDir)
	if err := store.EnsureDirs(); err != nil {
		t.Fatalf("EnsureDirs returned error: %v", err)
	}

	targetDir := store.ReleaseDir("v4.23.0")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatalf("create target dir failed: %v", err)
	}

	if err := store.SetLink(store.CurrentLinkPath(), targetDir); err != nil {
		t.Fatalf("SetLink returned error: %v", err)
	}

	resolved, err := store.LinkTarget(store.CurrentLinkPath())
	if err != nil {
		t.Fatalf("LinkTarget returned error: %v", err)
	}
	if resolved != filepath.Clean(targetDir) {
		t.Fatalf("unexpected link target: %s", resolved)
	}
}

func TestSanitizeVersion(t *testing.T) {
	sanitized := sanitizeVersion("../v4.23.0 test")
	if strings.Contains(sanitized, "..") || strings.Contains(sanitized, "/") || strings.Contains(sanitized, "\\") {
		t.Fatalf("unexpected sanitized version: %s", sanitized)
	}
}
