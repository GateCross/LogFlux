package ingest

import "time"

type LogIngestCursor struct {
	ID        uint `gorm:"primarykey"`
	CreatedAt time.Time
	UpdatedAt time.Time

	FilePath string `gorm:"size:1024;not null;uniqueIndex"`
	Offset   int64  `gorm:"not null;default:0"`
	// FileID 记录游标对应文件的 inode（unix 上由 dev+ino 拼成）。
	// 日志滚动后同名文件会被重建，inode 随之改变；此时旧偏移对新文件没有意义，
	// 必须从 0 重新采集。0 表示历史行，与当前 inode 比对时按「未知」处理。
	FileID uint64 `gorm:"not null;default:0"`
}

func (LogIngestCursor) TableName() string {
	return "log_ingest_cursors"
}
