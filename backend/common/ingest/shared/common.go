package shared

import (
	"encoding/json"
	"fmt"
	"math"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const defaultScanIntervalSec = 60

func DefaultScanIntervalSec() int {
	return defaultScanIntervalSec
}

func NormalizeSourceType(sourceType string) string {
	return strings.ToLower(strings.TrimSpace(sourceType))
}

func IsLogFileName(name string) bool {
	return strings.EqualFold(filepath.Ext(name), ".log")
}

// rolledLogSuffixRe 匹配 Caddy 日志滚动后的归档文件名。
//
// Caddy 用 timberjack 做滚动，归档名形如：
//
//	access-2026-10-26T16-00-00.000-size.log
//	access-2026-10-26T16-00-00.000-time.log
//	access-2026-10-26T16-00-00.000-size.log.gz
//
// 时间戳部分严格限定为 timberjack 默认布局 2006-01-02T15-04-05，
// 且为避免误杀 file-size.log / my-time.log 这类正常文件名，
// 必须要求「日期时间戳 + 原因」整体出现。
var rolledLogSuffixRe = regexp.MustCompile(
	`-\d{4}-\d{2}-\d{2}T\d{2}-\d{2}-\d{2}(?:\.\d+)?-(?:size|time)\.log(?:\.gz)?$`,
)

// IsRolledLogFileName 判断是否为滚动归档文件（含 gzip 压缩形态）。
func IsRolledLogFileName(name string) bool {
	return rolledLogSuffixRe.MatchString(name)
}

// IsActiveLogFileName 判断是否为应当采集的活动日志文件：
// 必须是 .log 后缀，且不是滚动归档。
func IsActiveLogFileName(name string) bool {
	if !IsLogFileName(name) {
		return false
	}
	return !IsRolledLogFileName(name)
}

func ParseUnixTS(value any) (time.Time, bool) {
	switch v := value.(type) {
	case json.Number:
		if f, err := v.Float64(); err == nil {
			return unixFloatToTime(f), true
		}
	case float64:
		return unixFloatToTime(v), true
	case int64:
		return time.Unix(v, 0), true
	case int:
		return time.Unix(int64(v), 0), true
	case string:
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return unixFloatToTime(f), true
		}
	}
	return time.Time{}, false
}

func unixFloatToTime(v float64) time.Time {
	sec, frac := math.Modf(v)
	return time.Unix(int64(sec), int64(frac*1e9))
}

func AsString(value any) string {
	if value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return v
	case json.Number:
		return v.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}

func AsFloat(value any) float64 {
	if value == nil {
		return 0
	}
	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case json.Number:
		if f, err := v.Float64(); err == nil {
			return f
		}
	case string:
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return 0
}
