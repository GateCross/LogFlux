package ingest

import (
	"strings"

	"logflux/model"

	"gorm.io/gorm"
)

// IngestManager 统一管理不同类型日志入库
type IngestManager struct {
	caddy  *CaddyIngestor
	system *SystemIngestor
}

func NewIngestManager(db *gorm.DB) *IngestManager {
	return &IngestManager{
		caddy:  NewCaddyIngestor(db),
		system: NewSystemIngestor(db),
	}
}

func (m *IngestManager) StartSource(source model.LogSource) {
	if !source.Enabled || strings.TrimSpace(source.Path) == "" {
		return
	}
	m.StartWithInterval(source.Path, source.ScanInterval, source.Type)
}

func (m *IngestManager) StopSource(source model.LogSource) {
	if strings.TrimSpace(source.Path) == "" {
		return
	}
	m.Stop(source.Path, source.Type)
}

func (m *IngestManager) StartWithInterval(path string, scanIntervalSec int, sourceType string) {
	switch normalizeSourceType(sourceType) {
	case "caddy":
		m.caddy.StartWithInterval(path, scanIntervalSec)
	case "caddy_runtime":
		m.system.StartWithInterval(path, scanIntervalSec, normalizeSourceType(sourceType))
	case "backend":
		// backend 日志直接写入数据库，不再从文件读取
		return
	default:
		// 未识别类型，默认按 caddy 访问日志处理（保持旧行为）
		m.caddy.StartWithInterval(path, scanIntervalSec)
	}
}

func (m *IngestManager) Stop(path string, sourceType string) {
	switch normalizeSourceType(sourceType) {
	case "caddy":
		m.caddy.Stop(path)
	case "caddy_runtime":
		m.system.Stop(path)
	case "backend":
		return
	default:
		m.caddy.Stop(path)
	}
}

func normalizeSourceType(sourceType string) string {
	return strings.ToLower(strings.TrimSpace(sourceType))
}
