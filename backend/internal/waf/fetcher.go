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
	ProxyURL       string
	TimeoutSec     int
	MaxBytes       int64
}

type FetchResult struct {
	SavedPath  string
	SizeBytes  int64
	StatusCode int
}

func FetchPackage(downloadURL, targetPath string, options FetchOptions) (*FetchResult, error) {
	if strings.TrimSpace(downloadURL) == "" {
		return nil, fmt.Errorf("下载地址不能为空")
	}
	if strings.TrimSpace(targetPath) == "" {
		return nil, fmt.Errorf("目标路径不能为空")
	}

	parsedURL, err := url.Parse(downloadURL)
	if err != nil {
		return nil, fmt.Errorf("URL 无效: %w", err)
	}
	if parsedURL.Scheme != "https" {
		return nil, fmt.Errorf("仅允许 HTTPS 协议")
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

	proxyURL := strings.TrimSpace(options.ProxyURL)
	if proxyURL != "" {
		parsedProxyURL, proxyErr := url.Parse(proxyURL)
		if proxyErr != nil {
			return nil, fmt.Errorf("代理 URL 无效: %w", proxyErr)
		}
		if parsedProxyURL.Scheme != "http" && parsedProxyURL.Scheme != "https" {
			return nil, fmt.Errorf("代理 URL 协议必须是 HTTP 或 HTTPS")
		}
		httpClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12},
			Proxy:           http.ProxyURL(parsedProxyURL),
		}
	}

	request, err := http.NewRequest(http.MethodGet, downloadURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	applyAuth(request, options.AuthType, options.AuthSecret)

	response, err := httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("获取资源失败: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("响应状态码异常: %d", response.StatusCode)
	}

	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return nil, fmt.Errorf("准备目标目录失败: %w", err)
	}

	tempFilePath := targetPath + ".part"
	tempFile, err := os.Create(tempFilePath)
	if err != nil {
		return nil, fmt.Errorf("创建临时文件失败: %w", err)
	}

	source := io.Reader(response.Body)
	if options.MaxBytes > 0 {
		source = &io.LimitedReader{R: response.Body, N: options.MaxBytes + 1}
	}
	writtenBytes, copyErr := io.Copy(tempFile, source)
	closeErr := tempFile.Close()
	if copyErr != nil {
		_ = os.Remove(tempFilePath)
		return nil, fmt.Errorf("写入临时文件失败: %w", copyErr)
	}
	if options.MaxBytes > 0 && writtenBytes > options.MaxBytes {
		_ = os.Remove(tempFilePath)
		return nil, fmt.Errorf("包文件过大: %d > %d", writtenBytes, options.MaxBytes)
	}
	if closeErr != nil {
		_ = os.Remove(tempFilePath)
		return nil, fmt.Errorf("关闭临时文件失败: %w", closeErr)
	}

	if err := os.Rename(tempFilePath, targetPath); err != nil {
		_ = os.Remove(tempFilePath)
		return nil, fmt.Errorf("移动临时文件失败: %w", err)
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
		return fmt.Errorf("主机名为空")
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
		if strings.HasSuffix(host, "."+normalizedAllowed) {
			return nil
		}
	}

	return fmt.Errorf("主机不在允许列表中: %s", host)
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
