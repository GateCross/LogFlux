package shared

import (
	"encoding/json"
	"fmt"
	"math"
	"path/filepath"
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
