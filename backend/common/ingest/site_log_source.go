package ingest

import (
	"errors"

	ingestmodel "logflux/model/ingest"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

// CaddySiteLogDir 是站点日志目录：每个站点在此下有独立的 <name>.log。
// 注册为目录源后，新增站点的日志文件会被采集器的 scanDir 自动发现。
const CaddySiteLogDir = "/var/log/caddy"

// EnsureSiteLogSource 确保站点日志目录已注册为日志源，使 Caddy 写入的
// 站点日志文件能被自动发现并采集。已存在时只做启用，不重复创建。
//
// 单实例改造后所有日志源都归唯一实例（server_id = 0），不再按节点隔离。
func EnsureSiteLogSource(db *gorm.DB, logger logx.Logger) {
	if db == nil {
		return
	}

	var source ingestmodel.LogSource
	err := db.Where("path = ? AND server_id = ?", CaddySiteLogDir, 0).First(&source).Error
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
		Name:         "站点日志目录",
		Path:         CaddySiteLogDir,
		Type:         "caddy",
		Enabled:      true,
		ScanInterval: 30,
		ServerID:     0,
	}
	if createErr := db.Create(&newSource).Error; createErr != nil {
		if logger != nil {
			logger.Errorf("注册站点日志源失败: %v", createErr)
		}
		return
	}
	if logger != nil {
		logger.Infof("已注册站点日志源: %s", CaddySiteLogDir)
	}
}
