package gorm

import (
	"context"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	gorm2 "gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/utils"
)

type LogxLogger struct {
	LogLevel                  logger.LogLevel
	SlowThreshold             time.Duration
	IgnoreRecordNotFoundError bool
}

func NewLogxLogger() *LogxLogger {
	return &LogxLogger{
		LogLevel:                  logger.Warn,
		SlowThreshold:             time.Second,
		IgnoreRecordNotFoundError: true,
	}
}

func (l *LogxLogger) LogMode(level logger.LogLevel) logger.Interface {
	newLogger := *l
	newLogger.LogLevel = level
	return &newLogger
}

func (l *LogxLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= logger.Info {
		logx.WithContext(ctx).Infof(msg, data...)
	}
}

func (l *LogxLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= logger.Warn {
		// go-zero logx doesn't have Warn, map to Error but maybe with a prefix or just Error
		logx.WithContext(ctx).Errorf("[WARN] "+msg, data...)
	}
}

func (l *LogxLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= logger.Error {
		logx.WithContext(ctx).Errorf(msg, data...)
	}
}

func (l *LogxLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.LogLevel <= logger.Silent {
		return
	}

	elapsed := time.Since(begin)
	switch {
	case err != nil && l.LogLevel >= logger.Error && (!l.IgnoreRecordNotFoundError || err != gorm2.ErrRecordNotFound):
		sql, rows := fc()
		fileWithLineNum := utils.FileWithLineNum()
		logx.WithContext(ctx).Errorf("%s | [%.3fms] [rows:%v] %s | Error: %v", fileWithLineNum, float64(elapsed.Nanoseconds())/1e6, rows, sql, err)
	case l.SlowThreshold != 0 && elapsed > l.SlowThreshold && l.LogLevel >= logger.Warn:
		sql, rows := fc()
		fileWithLineNum := utils.FileWithLineNum()
		logx.WithContext(ctx).Slowf("%s | [%.3fms] [rows:%v] %s | SLOW SQL >= %v", fileWithLineNum, float64(elapsed.Nanoseconds())/1e6, rows, sql, l.SlowThreshold)
	case l.LogLevel >= logger.Info:
		sql, rows := fc()
		fileWithLineNum := utils.FileWithLineNum()
		logx.WithContext(ctx).Infof("%s | [%.3fms] [rows:%v] %s", fileWithLineNum, float64(elapsed.Nanoseconds())/1e6, rows, sql)
	}
}
