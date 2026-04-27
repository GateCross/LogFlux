package logger

import (
	"context"
	"fmt"

	"github.com/zeromicro/go-zero/core/logx"
)

type Module string

const (
	ModuleSystem       Module = "system"
	ModuleMedia        Module = "media"
	ModuleSite         Module = "site"
	ModuleTorrent      Module = "torrent"
	ModuleSubscribe    Module = "subscribe"
	ModuleTransfer     Module = "transfer"
	ModuleCloudStorage Module = "cloud_storage"
	ModulePlugin       Module = "plugin"
	ModuleMlist        Module = "mlist"
	ModuleCaddy        Module = "caddy"
	ModuleLog          Module = "log"
	ModuleCron         Module = "cron"
	ModuleNotification Module = "notification"
	ModuleUser         Module = "user"
)

// Logger 是项目统一日志接口的兼容别名。
type Logger = logx.Logger

type moduleLogger struct {
	module Module
}

// New 创建指定模块的日志构造器。
func New(module Module) *moduleLogger {
	return &moduleLogger{module: module}
}

// WithContext 返回带上下文的 go-zero logger，并自动补充模块字段。
func (l *moduleLogger) WithContext(ctx context.Context) logx.Logger {
	base := logx.WithContext(ctx)
	if l == nil || l.module == "" {
		return base
	}
	return base.WithFields(logx.Field("module", string(l.module)))
}

// Errorc 输出带上下文的中文错误日志。
func Errorc(ctx context.Context, err error) {
	if err == nil {
		return
	}
	logx.WithContext(ctx).Errorf("请求处理失败: %v", err)
}

// Errorcf 输出带上下文的中文格式化错误日志。
func Errorcf(ctx context.Context, err error, format string, args ...interface{}) {
	if err == nil {
		return
	}
	msg := fmt.Sprintf(format, args...)
	logx.WithContext(ctx).Errorf("%s: %v", msg, err)
}
