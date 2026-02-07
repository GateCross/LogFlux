package utils

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ParseOptionalTime 解析时间字符串或 Unix 时间戳为本地时间。
// 传入空字符串时返回 (nil, nil)。
func ParseOptionalTime(value string) (*time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}

	if t, ok := parseUnixTimestamp(value); ok {
		return t, nil
	}

	layouts := []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, value, time.Local); err == nil {
			return &t, nil
		}
	}
	return nil, fmt.Errorf("unsupported time format")
}

func parseUnixTimestamp(value string) (*time.Time, bool) {
	isDigits := true
	for _, ch := range value {
		if ch < '0' || ch > '9' {
			isDigits = false
			break
		}
	}
	if !isDigits || len(value) < 10 {
		return nil, false
	}

	ts, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return nil, false
	}

	var t time.Time
	switch {
	case len(value) >= 19:
		t = time.Unix(0, ts)
	case len(value) >= 16:
		t = time.Unix(0, ts*int64(time.Microsecond))
	case len(value) >= 13:
		t = time.Unix(0, ts*int64(time.Millisecond))
	default:
		t = time.Unix(ts, 0)
	}

	local := t.In(time.Local)
	return &local, true
}
