package waf

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVerifyPackage_Success(t *testing.T) {
	tempDir := t.TempDir()
	packagePath := filepath.Join(tempDir, "rules.tar.gz")
	if err := os.WriteFile(packagePath, []byte("test-package-content"), 0o644); err != nil {
		t.Fatalf("write package failed: %v", err)
	}

	result, err := VerifyPackage(packagePath, VerifyOptions{MaxPackageBytes: 1024})
	if err != nil {
		t.Fatalf("VerifyPackage returned error: %v", err)
	}
	if result.Ext != ".tar.gz" {
		t.Fatalf("expected ext .tar.gz, got %s", result.Ext)
	}
	if result.SizeBytes == 0 {
		t.Fatalf("expected size > 0")
	}
	if len(result.SHA256) != 64 {
		t.Fatalf("expected sha256 len 64, got %d", len(result.SHA256))
	}
}

func TestVerifyPackage_UnsupportedExt(t *testing.T) {
	tempDir := t.TempDir()
	packagePath := filepath.Join(tempDir, "rules.txt")
	if err := os.WriteFile(packagePath, []byte("test"), 0o644); err != nil {
		t.Fatalf("write package failed: %v", err)
	}

	_, err := VerifyPackage(packagePath, VerifyOptions{})
	if err == nil || !strings.Contains(err.Error(), "unsupported package extension") {
		t.Fatalf("expected unsupported ext error, got %v", err)
	}
}

func TestVerifyPackage_SHA256Mismatch(t *testing.T) {
	tempDir := t.TempDir()
	packagePath := filepath.Join(tempDir, "rules.zip")
	if err := os.WriteFile(packagePath, []byte("zip-content"), 0o644); err != nil {
		t.Fatalf("write package failed: %v", err)
	}

	_, err := VerifyPackage(packagePath, VerifyOptions{ExpectedSHA256: strings.Repeat("a", 64)})
	if err == nil || !strings.Contains(err.Error(), "sha256 mismatch") {
		t.Fatalf("expected sha mismatch error, got %v", err)
	}
}

func TestVerifyPackage_TooLarge(t *testing.T) {
	tempDir := t.TempDir()
	packagePath := filepath.Join(tempDir, "rules.zip")
	if err := os.WriteFile(packagePath, []byte("123456789"), 0o644); err != nil {
		t.Fatalf("write package failed: %v", err)
	}

	_, err := VerifyPackage(packagePath, VerifyOptions{MaxPackageBytes: 4})
	if err == nil || !strings.Contains(err.Error(), "package too large") {
		t.Fatalf("expected too large error, got %v", err)
	}
}
