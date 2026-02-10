package waf

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	DefaultFetchTimeoutSec = 60
)

type FetchOptions struct {
	AllowedDomains []string
	AuthType       string
	AuthSecret     string
	TimeoutSec     int
}

type FetchResult struct {
	SavedPath  string
	SizeBytes  int64
	StatusCode int
}

func FetchPackage(downloadURL, targetPath string, options FetchOptions) (*FetchResult, error) {
	if strings.TrimSpace(downloadURL) == "" {
		return nil, fmt.Errorf("download url is required")
	}
	if strings.TrimSpace(targetPath) == "" {
		return nil, fmt.Errorf("target path is required")
	}

	parsedURL, err := url.Parse(downloadURL)
	if err != nil {
		return nil, fmt.Errorf("invalid url: %w", err)
	}
	if parsedURL.Scheme != "https" {
		return nil, fmt.Errorf("only https scheme is allowed")
	}
	if err := validateDomainAllowlist(parsedURL.Hostname(), options.AllowedDomains); err != nil {
		return nil, err
	}

	timeout := time.Duration(options.TimeoutSec) * time.Second
	if timeout <= 0 {
		timeout = DefaultFetchTimeoutSec * time.Second
	}

	httpClient := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12},
		},
	}

	request, err := http.NewRequest(http.MethodGet, downloadURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}
	applyAuth(request, options.AuthType, options.AuthSecret)

	response, err := httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("fetch failed: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status code: %d", response.StatusCode)
	}

	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return nil, fmt.Errorf("prepare target dir failed: %w", err)
	}

	tempFilePath := targetPath + ".part"
	tempFile, err := os.Create(tempFilePath)
	if err != nil {
		return nil, fmt.Errorf("create temp file failed: %w", err)
	}

	writtenBytes, copyErr := io.Copy(tempFile, response.Body)
	closeErr := tempFile.Close()
	if copyErr != nil {
		_ = os.Remove(tempFilePath)
		return nil, fmt.Errorf("write temp file failed: %w", copyErr)
	}
	if closeErr != nil {
		_ = os.Remove(tempFilePath)
		return nil, fmt.Errorf("close temp file failed: %w", closeErr)
	}

	if err := os.Rename(tempFilePath, targetPath); err != nil {
		_ = os.Remove(tempFilePath)
		return nil, fmt.Errorf("move temp file failed: %w", err)
	}

	return &FetchResult{
		SavedPath:  targetPath,
		SizeBytes:  writtenBytes,
		StatusCode: response.StatusCode,
	}, nil
}

func validateDomainAllowlist(host string, allowlist []string) error {
	host = strings.ToLower(strings.TrimSpace(host))
	if host == "" {
		return fmt.Errorf("host is empty")
	}

	if len(allowlist) == 0 {
		return nil
	}

	for _, allowedDomain := range allowlist {
		normalizedAllowed := strings.ToLower(strings.TrimSpace(allowedDomain))
		if normalizedAllowed == "" {
			continue
		}
		if host == normalizedAllowed {
			return nil
		}
	}

	return fmt.Errorf("host not allowed: %s", host)
}

func applyAuth(request *http.Request, authType, authSecret string) {
	switch strings.ToLower(strings.TrimSpace(authType)) {
	case "token":
		secret := strings.TrimSpace(authSecret)
		if secret != "" {
			request.Header.Set("Authorization", "Bearer "+secret)
		}
	case "basic":
		parts := strings.SplitN(authSecret, ":", 2)
		if len(parts) == 2 {
			request.SetBasicAuth(parts[0], parts[1])
		}
	}
}
