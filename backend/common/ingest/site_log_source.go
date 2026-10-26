package ingest

import (
	"errors"
	"fmt"

	ingestmodel "logflux/model/ingest"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

// CaddySiteLogDir 是站点日志目录：每个站点在此下有独立的 <name>.log。
// 注册为目录源后，新增站点的日志文件会被采集器的 scanDir 自动发现。
const CaddySiteLogDir = "/var/log/caddy"

// EnsureSiteLogSource 确保站点日志目录已注册为日志源，使 Caddy 写入的
// 站点日志文件能被自动发现并采集。已存在时只做启用，不重复创建。
func EnsureSiteLogSource(db *gorm.DB, serverID uint, logger logx.Logger) {
	if db == nil {
		return
	}

	var source ingestmodel.LogSource
	err := db.Where("path = ? AND server_id = ?", CaddySiteLogDir, serverID).First(&source).Error
	if err == nil {
		if !source.Enabled {
			db.Model(&ingestmodel.LogSource{}).Where("id = ?", source.ID).Update("enabled", true)
		}
		return
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		if logger != nil {
			logger.Errorf("查询站点日志源失败: %v", err)
		}
		return
	}

	newSource := ingestmodel.LogSource{
		Name:         fmt.Sprintf("站点日志目录（节点 %d）", serverID),
		Path:         CaddySiteLogDir,
		Type:         "caddy",
		Enabled:      true,
		ScanInterval: 30,
		ServerID:     serverID,
	}
	if createErr := db.Create(&newSource).Error; createErr != nil {
		if logger != nil {
			logger.Errorf("注册站点日志源失败: %v", createErr)
		}
		return
	}
	if logger != nil {
		logger.Infof("已注册站点日志源: %s (节点 %d)", CaddySiteLogDir, serverID)
	}
}
