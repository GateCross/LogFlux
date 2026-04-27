package log

import "fmt"

var errInvalidLogSourcePath = fmt.Errorf("日志源路径不能为空")
var errInvalidLogSourceScanInterval = fmt.Errorf("扫描间隔必须大于 0 秒（默认 60）")
