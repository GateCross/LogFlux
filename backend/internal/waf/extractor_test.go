package waf

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractPackage_ZipSuccess(t *testing.T) {
	tempDir := t.TempDir()
	zipPath := filepath.Join(tempDir, "rules.zip")
	if err := createZip(zipPath, map[string]string{"rules/test.conf": "SecRuleEngine On"}); err != nil {
		t.Fatalf("create zip failed: %v", err)
	}

	targetDir := filepath.Join(tempDir, "out")
	result, err := ExtractPackage(zipPath, targetDir, ExtractOptions{})
	if err != nil {
		t.Fatalf("ExtractPackage returned error: %v", err)
	}
	if result.FileCount != 1 {
		t.Fatalf("expected file count 1, got %d", result.FileCount)
	}

	extractedFile := filepath.Join(targetDir, "rules/test.conf")
	if _, err := os.Stat(extractedFile); err != nil {
		t.Fatalf("expected extracted file exists: %v", err)
	}
}

func TestExtractPackage_ZipSlipBlocked(t *testing.T) {
	tempDir := t.TempDir()
	zipPath := filepath.Join(tempDir, "bad.zip")
	if err := createZip(zipPath, map[string]string{"../../evil.txt": "bad"}); err != nil {
		t.Fatalf("create zip failed: %v", err)
	}

	_, err := ExtractPackage(zipPath, filepath.Join(tempDir, "out"), ExtractOptions{})
	if err == nil || !strings.Contains(err.Error(), "path traversal") {
		t.Fatalf("expected path traversal error, got %v", err)
	}
}

func TestExtractPackage_TarGzSuccess(t *testing.T) {
	tempDir := t.TempDir()
	tarPath := filepath.Join(tempDir, "rules.tar.gz")
	if err := createTarGz(tarPath, map[string]string{"rules/base.conf": "SecRuleEngine On"}); err != nil {
		t.Fatalf("create tar.gz failed: %v", err)
	}

	targetDir := filepath.Join(tempDir, "out")
	result, err := ExtractPackage(tarPath, targetDir, ExtractOptions{})
	if err != nil {
		t.Fatalf("ExtractPackage returned error: %v", err)
	}
	if result.FileCount != 1 {
		t.Fatalf("expected file count 1, got %d", result.FileCount)
	}
}

func createZip(zipPath string, files map[string]string) error {
	zipFile, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	writer := zip.NewWriter(zipFile)
	defer writer.Close()

	for name, content := range files {
		entryWriter, err := writer.Create(name)
		if err != nil {
			return err
		}
		if _, err := entryWriter.Write([]byte(content)); err != nil {
			return err
		}
	}
	return nil
}

func createTarGz(tarPath string, files map[string]string) error {
	targetFile, err := os.Create(tarPath)
	if err != nil {
		return err
	}
	defer targetFile.Close()

	gzipWriter := gzip.NewWriter(targetFile)
	defer gzipWriter.Close()

	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	for name, content := range files {
		header := &tar.Header{
			Name: name,
			Mode: 0o644,
			Size: int64(len(content)),
		}
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}
		if _, err := tarWriter.Write([]byte(content)); err != nil {
			return err
		}
	}

	return nil
}
