package caddy

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"logflux/internal/svc"
	"logflux/model"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

const (
	caddyRequestTimeout = 8 * time.Second
	caddyMaxRetries     = 2
)

func adaptCaddyfile(server *model.CaddyServer, config string) error {
	_, _, err := postCaddyText(server, "/adapt", "text/caddyfile", config)
	if err != nil {
		return fmt.Errorf("adapt failed: %w", err)
	}
	return nil
}

func loadCaddyfile(server *model.CaddyServer, config string) error {
	_, _, err := postCaddyText(server, "/load", "text/caddyfile", config)
	if err != nil {
		return fmt.Errorf("load failed: %w", err)
	}
	return nil
}

func postCaddyText(server *model.CaddyServer, endpoint, contentType, body string) (int, []byte, error) {
	var lastErr error
	for attempt := 0; attempt < caddyMaxRetries; attempt++ {
		req, err := http.NewRequest("POST", strings.TrimRight(server.Url, "/")+endpoint, bytes.NewBufferString(body))
		if err != nil {
			return 0, nil, err
		}
		req.Header.Set("Content-Type", contentType)
		if server.Token != "" {
			req.Header.Set("Authorization", "Bearer "+server.Token)
		}

		client := &http.Client{Timeout: caddyRequestTimeout}
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			time.Sleep(time.Duration(attempt+1) * 300 * time.Millisecond)
			continue
		}
		defer resp.Body.Close()
		respBody, _ := io.ReadAll(resp.Body)
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return resp.StatusCode, respBody, nil
		}
		return resp.StatusCode, respBody, fmt.Errorf("caddy api error: %s", strings.TrimSpace(string(respBody)))
	}
	return 0, nil, lastErr
}

func getCaddyConfigJSON(server *model.CaddyServer) ([]byte, error) {
	req, err := http.NewRequest("GET", strings.TrimRight(server.Url, "/")+"/config/", nil)
	if err != nil {
		return nil, err
	}
	if server.Token != "" {
		req.Header.Set("Authorization", "Bearer "+server.Token)
	}
	client := &http.Client{Timeout: caddyRequestTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("caddy config fetch failed: %s", strings.TrimSpace(string(body)))
	}
	return io.ReadAll(resp.Body)
}

func hashConfig(config string) string {
	sum := sha256.Sum256([]byte(config))
	return hex.EncodeToString(sum[:])
}

func syncCaddyLogSources(svcCtx *svc.ServiceContext, server *model.CaddyServer, logger logx.Logger) {
	body, err := getCaddyConfigJSON(server)
	if err != nil {
		logger.Errorf("同步日志配置失败: %v", err)
		return
	}

	paths := discoverLogPathsFromConfigJSON(body)
	pathSet := make(map[string]struct{}, len(paths))
	for _, path := range paths {
		if path == "" {
			continue
		}
		pathSet[path] = struct{}{}
	}

	var autoSources []model.LogSource
	svcCtx.DB.Where("type = ? AND name LIKE ?", "caddy", "Caddy Auto:%").Find(&autoSources)
	for _, source := range autoSources {
		if _, ok := pathSet[source.Path]; !ok {
			svcCtx.DB.Model(&model.LogSource{}).Where("id = ?", source.ID).Update("enabled", false)
			svcCtx.Ingestor.Stop(source.Path)
		}
	}

	for path := range pathSet {
		var source model.LogSource
		err := svcCtx.DB.Where("path = ?", path).First(&source).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			newSource := model.LogSource{
				Name:    fmt.Sprintf("Caddy Auto: %s", path),
				Path:    path,
				Type:    "caddy",
				Enabled: true,
			}
			if err := svcCtx.DB.Create(&newSource).Error; err == nil {
				logger.Infof("自动添加日志源: %s", path)
				svcCtx.Ingestor.Start(path)
			}
			continue
		}
		if err != nil {
			logger.Errorf("查询日志源失败: %v", err)
			continue
		}
		if !source.Enabled {
			svcCtx.DB.Model(&model.LogSource{}).Where("id = ?", source.ID).Update("enabled", true)
		}
		svcCtx.Ingestor.Start(path)
	}
}

func discoverLogPathsFromConfigJSON(raw []byte) []string {
	var data any
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil
	}
	results := make(map[string]struct{})
	var walk func(node any)
	walk = func(node any) {
		switch v := node.(type) {
		case map[string]any:
			if output, ok := v["output"].(string); ok && output == "file" {
				if filename, ok := v["filename"].(string); ok && filename != "" {
					results[filename] = struct{}{}
				}
			}
			for _, child := range v {
				walk(child)
			}
		case []any:
			for _, child := range v {
				walk(child)
			}
		}
	}
	walk(data)

	paths := make([]string, 0, len(results))
	for path := range results {
		paths = append(paths, path)
	}
	return paths
}
