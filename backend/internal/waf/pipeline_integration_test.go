package waf

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type integrationLoader struct {
	adaptShouldFail bool
	loadShouldFail  bool
}

func (loader *integrationLoader) Adapt(string) error {
	if loader.adaptShouldFail {
		return fmt.Errorf("adapt failed")
	}
	return nil
}

func (loader *integrationLoader) Load(string) error {
	if loader.loadShouldFail {
		return fmt.Errorf("load failed")
	}
	return nil
}

func TestFetchVerifyActivateRollbackPipeline(t *testing.T) {
	baseDir := t.TempDir()
	store := NewStore(baseDir)
	if err := store.EnsureDirs(); err != nil {
		t.Fatalf("ensure dirs failed: %v", err)
	}

	previousDir := store.ReleaseDir("v1.0.0")
	if err := os.MkdirAll(previousDir, 0o755); err != nil {
		t.Fatalf("prepare previous release dir failed: %v", err)
	}
	if err := store.SetLink(store.CurrentLinkPath(), previousDir); err != nil {
		t.Fatalf("set current link failed: %v", err)
	}

	packageBytes := buildTestTarGz(t, map[string]string{
		"coraza.conf":        "SecRuleEngine On\n",
		"rules/example.conf": "SecRule ARGS test",
		"plugins/readme.txt": "ok",
	})
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(packageBytes)
	}))
	defer server.Close()

	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse server url failed: %v", err)
	}

	targetPath := store.StagePath("integration.tar.gz")
	fetchResult, err := fetchWithTestTLS(server.Client(), server.URL, targetPath, FetchOptions{
		AllowedDomains: []string{serverURL.Hostname()},
		TimeoutSec:     10,
	})
	if err != nil {
		t.Fatalf("fetch package failed: %v", err)
	}
	if fetchResult.SizeBytes <= 0 {
		t.Fatalf("unexpected fetched size: %d", fetchResult.SizeBytes)
	}

	verifyResult, err := VerifyPackage(fetchResult.SavedPath, VerifyOptions{
		AllowedExt:      []string{".tar.gz", ".zip"},
		MaxPackageBytes: 10 * 1024 * 1024,
	})
	if err != nil {
		t.Fatalf("verify package failed: %v", err)
	}
	if verifyResult.Ext != ".tar.gz" {
		t.Fatalf("unexpected ext: %s", verifyResult.Ext)
	}

	releaseDir := store.ReleaseDir("v2.0.0")
	if err := os.MkdirAll(releaseDir, 0o755); err != nil {
		t.Fatalf("create release dir failed: %v", err)
	}
	if _, err := ExtractPackage(fetchResult.SavedPath, releaseDir, ExtractOptions{
		MaxFiles:      100,
		MaxTotalBytes: 10 * 1024 * 1024,
	}); err != nil {
		t.Fatalf("extract package failed: %v", err)
	}

	loader := &integrationLoader{adaptShouldFail: true}
	activator := &Activator{
		Store:       store,
		CaddyLoader: loader,
	}
	if err := activator.ActivateVersion("v2.0.0", "test-config"); err == nil {
		t.Fatalf("expect activate to fail for rollback")
	}

	currentAfterRollback, err := store.LinkTarget(store.CurrentLinkPath())
	if err != nil {
		t.Fatalf("read current link after rollback failed: %v", err)
	}
	if filepath.Clean(currentAfterRollback) != filepath.Clean(previousDir) {
		t.Fatalf("current link should rollback to previous: got=%s want=%s", currentAfterRollback, previousDir)
	}

	loader.adaptShouldFail = false
	if err := activator.ActivateVersion("v2.0.0", "test-config"); err != nil {
		t.Fatalf("activate version failed: %v", err)
	}

	currentTarget, err := store.LinkTarget(store.CurrentLinkPath())
	if err != nil {
		t.Fatalf("read current link failed: %v", err)
	}
	if filepath.Clean(currentTarget) != filepath.Clean(releaseDir) {
		t.Fatalf("unexpected current target: got=%s want=%s", currentTarget, releaseDir)
	}

	lastGoodTarget, err := store.LinkTarget(store.LastGoodLinkPath())
	if err != nil {
		t.Fatalf("read last_good link failed: %v", err)
	}
	if filepath.Clean(lastGoodTarget) != filepath.Clean(previousDir) {
		t.Fatalf("unexpected last_good target: got=%s want=%s", lastGoodTarget, previousDir)
	}
}

func fetchWithTestTLS(client *http.Client, downloadURL, targetPath string, options FetchOptions) (*FetchResult, error) {
	if client == nil {
		return FetchPackage(downloadURL, targetPath, options)
	}

	parsedURL, err := url.Parse(downloadURL)
	if err != nil {
		return nil, err
	}
	if err := validateDomainAllowlist(parsedURL.Hostname(), options.AllowedDomains); err != nil {
		return nil, err
	}

	timeout := DefaultFetchTimeoutSec
	if options.TimeoutSec > 0 {
		timeout = options.TimeoutSec
	}
	client.Timeout = time.Duration(timeout) * time.Second
	client.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // #nosec G402 测试环境使用自签证书
	}

	request, err := http.NewRequest(http.MethodGet, downloadURL, nil)
	if err != nil {
		return nil, err
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status code: %d", response.StatusCode)
	}

	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return nil, err
	}
	file, err := os.Create(targetPath)
	if err != nil {
		return nil, err
	}
	writtenBytes, copyErr := io.Copy(file, response.Body)
	closeErr := file.Close()
	if copyErr != nil {
		_ = os.Remove(targetPath)
		return nil, copyErr
	}
	if closeErr != nil {
		_ = os.Remove(targetPath)
		return nil, closeErr
	}
	return &FetchResult{
		SavedPath:  targetPath,
		SizeBytes:  writtenBytes,
		StatusCode: response.StatusCode,
	}, nil
}

func buildTestTarGz(t *testing.T, files map[string]string) []byte {
	t.Helper()

	var buffer bytes.Buffer
	gzipWriter := gzip.NewWriter(&buffer)
	tarWriter := tar.NewWriter(gzipWriter)

	for name, content := range files {
		payload := []byte(content)
		header := &tar.Header{
			Name: name,
			Mode: 0o644,
			Size: int64(len(payload)),
		}
		if err := tarWriter.WriteHeader(header); err != nil {
			t.Fatalf("write tar header failed: %v", err)
		}
		if _, err := tarWriter.Write(payload); err != nil {
			t.Fatalf("write tar payload failed: %v", err)
		}
	}

	if err := tarWriter.Close(); err != nil {
		t.Fatalf("close tar writer failed: %v", err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatalf("close gzip writer failed: %v", err)
	}
	return buffer.Bytes()
}
