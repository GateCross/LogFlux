package ingest

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"logflux/model"

	"github.com/nxadm/tail"
	"github.com/zeromicro/go-zero/core/logx"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Log Format:
// [{ts}] "{country_name}" "{province_name}" "{city_name}" "{request>host}" "{request>method} {request>uri} {request>proto}" {status} {size} "{request>headers>User-Agent>[0]}" "{request>remote_ip}" "{request>client_ip}"

var logRegex = regexp.MustCompile(`^\[(.*?)\] "(.*?)" "(.*?)" "(.*?)" "(.*?)" "(.*?) (.*?) (.*?)" (\d+) (\d+) "(.*?)" "(.*?)" "(.*?)"$`)

const defaultScanIntervalSec = 60

type dirWatcher struct {
	stopCh   chan struct{}
	interval time.Duration
}

type CaddyIngestor struct {
	db          *gorm.DB
	tails       map[string]*tail.Tail
	dirWatchers map[string]dirWatcher
	dirFiles    map[string]map[string]struct{}
	mu          sync.Mutex
}

func NewCaddyIngestor(db *gorm.DB) *CaddyIngestor {
	return &CaddyIngestor{
		db:          db,
		tails:       make(map[string]*tail.Tail),
		dirWatchers: make(map[string]dirWatcher),
		dirFiles:    make(map[string]map[string]struct{}),
	}
}

func (i *CaddyIngestor) ParseLine(line string) (*model.CaddyLog, error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil, fmt.Errorf("empty line")
	}

	if strings.HasPrefix(line, "{") {
		if logEntry, err := i.parseJSONLine(line); err == nil {
			return logEntry, nil
		}
	}

	matches := logRegex.FindStringSubmatch(line)
	if len(matches) != 14 {
		return nil, fmt.Errorf("invalid log format: %s", line)
	}

	logTime, err := i.parseTime(matches[1])
	if err != nil {
		logx.Errorf("Time parse error: %v for %s", err, matches[1])
	}

	status, _ := strconv.Atoi(matches[9])
	size, _ := strconv.ParseInt(matches[10], 10, 64)

	return &model.CaddyLog{
		LogTime:   logTime,
		Country:   matches[2],
		Province:  matches[3],
		City:      matches[4],
		Host:      matches[5],
		Method:    matches[6],
		Uri:       matches[7],
		Proto:     matches[8],
		Status:    status,
		Size:      size,
		UserAgent: matches[11],
		RemoteIP:  matches[12],
		ClientIP:  matches[13],
		RawLog:    mustJSONRaw(line),
		ExtraData: "{}",
	}, nil
}

func (i *CaddyIngestor) parseTime(ts string) (time.Time, error) {
	layouts := []string{
		"2006/01/02 15:04:05.000",
		"02/Jan/2006:15:04:05 -0700",
		"2006-01-02 15:04:05",
		time.RFC3339,
	}

	for _, layout := range layouts {
		if t, err := time.Parse(layout, ts); err == nil {
			return t, nil
		}
		if t, err := time.ParseInLocation(layout, ts, time.Local); err == nil {
			return t, nil
		}
	}
	return time.Now(), fmt.Errorf("unknown time format")
}

func (i *CaddyIngestor) Ingest(line string) error {
	logEntry, err := i.ParseLine(line)
	if err != nil {
		return err
	}
	if err := i.db.Create(logEntry).Error; err != nil {
		logx.Errorf("DB insert failed: %v", err)
		return err
	}
	return nil
}

func (i *CaddyIngestor) Start(filePath string) {
	i.StartWithInterval(filePath, 0)
}

func (i *CaddyIngestor) StartWithInterval(filePath string, scanIntervalSec int) {
	filePath = strings.TrimSpace(filePath)
	if filePath == "" {
		return
	}
	filePath = filepath.Clean(filePath)

	if info, err := os.Stat(filePath); err == nil && info.IsDir() {
		i.startDir(filePath, scanIntervalSec)
		return
	}

	i.startFile(filePath)
}

func (i *CaddyIngestor) startFile(filePath string) bool {
	i.mu.Lock()
	if _, exists := i.tails[filePath]; exists {
		i.mu.Unlock()
		return false
	}
	i.mu.Unlock()

	startOffset := i.resolveStartOffset(filePath)

	t, err := tail.TailFile(filePath, tail.Config{
		Follow:   true,
		ReOpen:   true,
		Poll:     true,
		Location: &tail.SeekInfo{Offset: startOffset, Whence: io.SeekStart},
	})
	if err != nil {
		logx.Errorf("Error tailing file: %v", err)
		return false
	}

	i.mu.Lock()
	if _, exists := i.tails[filePath]; exists {
		i.mu.Unlock()
		t.Stop()
		t.Cleanup()
		return false
	}
	i.tails[filePath] = t
	i.mu.Unlock()

	logx.Infof("Started monitoring: %s", filePath)

	go func(path string) {
		for line := range t.Lines {
			if line == nil {
				continue
			}
			if line.Err != nil {
				logx.Errorf("Tail read failed: %v", line.Err)
				continue
			}
			if err := i.Ingest(line.Text); err != nil {
				// keep noisy errors in stdout for now
				logx.Errorf("Log ingest failed: %v", err)
				continue
			}
			if err := i.saveOffset(path, line.SeekInfo.Offset); err != nil {
				logx.Errorf("Save ingest cursor failed: %v", err)
			}
		}
	}(filePath)

	return true
}

func (i *CaddyIngestor) resolveStartOffset(filePath string) int64 {
	var cursor model.LogIngestCursor
	if err := i.db.Where("file_path = ?", filePath).Take(&cursor).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			logx.Errorf("Load ingest cursor failed: %v", err)
		}
		return 0
	}

	offset := cursor.Offset
	if offset < 0 {
		return 0
	}

	info, err := os.Stat(filePath)
	if err != nil {
		return offset
	}
	if offset > info.Size() {
		return 0
	}

	return offset
}

func (i *CaddyIngestor) saveOffset(filePath string, offset int64) error {
	if offset < 0 {
		offset = 0
	}

	cursor := model.LogIngestCursor{
		FilePath: filePath,
		Offset:   offset,
	}

	return i.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "file_path"}},
		DoUpdates: clause.Assignments(map[string]any{
			"offset":     offset,
			"updated_at": time.Now(),
		}),
	}).Create(&cursor).Error
}

func (i *CaddyIngestor) startDir(dirPath string, scanIntervalSec int) {
	if scanIntervalSec <= 0 {
		scanIntervalSec = defaultScanIntervalSec
	}
	interval := time.Duration(scanIntervalSec) * time.Second

	var oldStopCh chan struct{}
	i.mu.Lock()
	if watcher, exists := i.dirWatchers[dirPath]; exists {
		if watcher.interval == interval {
			i.mu.Unlock()
			return
		}
		oldStopCh = watcher.stopCh
	}
	stopCh := make(chan struct{})
	i.dirWatchers[dirPath] = dirWatcher{stopCh: stopCh, interval: interval}
	if _, ok := i.dirFiles[dirPath]; !ok {
		i.dirFiles[dirPath] = make(map[string]struct{})
	}
	i.mu.Unlock()

	if oldStopCh != nil {
		close(oldStopCh)
	}

	i.scanDir(dirPath)

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				i.scanDir(dirPath)
			case <-stopCh:
				return
			}
		}
	}()
}

func (i *CaddyIngestor) scanDir(dirPath string) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		logx.Errorf("Error reading dir: %v", err)
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !isLogFileName(name) {
			continue
		}
		filePath := filepath.Join(dirPath, name)

		i.mu.Lock()
		dirFiles, ok := i.dirFiles[dirPath]
		if !ok {
			i.mu.Unlock()
			return
		}
		_, tracked := dirFiles[filePath]
		i.mu.Unlock()
		if tracked {
			continue
		}

		if i.startFile(filePath) {
			i.mu.Lock()
			if dirFiles, ok := i.dirFiles[dirPath]; ok {
				dirFiles[filePath] = struct{}{}
			}
			i.mu.Unlock()
		}
	}
}

func (i *CaddyIngestor) Stop(filePath string) {
	filePath = strings.TrimSpace(filePath)
	if filePath == "" {
		return
	}
	filePath = filepath.Clean(filePath)

	i.mu.Lock()
	watcher, isDir := i.dirWatchers[filePath]
	files := i.dirFiles[filePath]
	if isDir {
		delete(i.dirWatchers, filePath)
		delete(i.dirFiles, filePath)
	}
	i.mu.Unlock()

	if isDir {
		close(watcher.stopCh)
		for file := range files {
			i.stopFile(file)
		}
		return
	}

	i.stopFile(filePath)
}

func (i *CaddyIngestor) stopFile(filePath string) {
	i.mu.Lock()
	t, exists := i.tails[filePath]
	if exists {
		delete(i.tails, filePath)
	}
	i.mu.Unlock()

	if exists {
		t.Stop()
		t.Cleanup()
		logx.Infof("Stopped monitoring: %s", filePath)
	}
}

func isLogFileName(name string) bool {
	return strings.EqualFold(filepath.Ext(name), ".log")
}

func DefaultScanIntervalSec() int {
	return defaultScanIntervalSec
}

func (i *CaddyIngestor) parseJSONLine(line string) (*model.CaddyLog, error) {
	decoder := json.NewDecoder(strings.NewReader(line))
	decoder.UseNumber()

	var data map[string]any
	if err := decoder.Decode(&data); err != nil {
		return nil, err
	}

	entry := &model.CaddyLog{
		RawLog:    line,
		ExtraData: "{}",
	}

	if ts, ok := parseUnixTS(data["ts"]); ok {
		entry.LogTime = ts
	} else {
		entry.LogTime = time.Now()
	}

	entry.Status = int(asFloat(data["status"]))
	entry.Size = int64(asFloat(data["size"]))

	if req, ok := data["request"].(map[string]any); ok {
		entry.Host = asString(req["host"])
		entry.Method = asString(req["method"])
		entry.Uri = asString(req["uri"])
		entry.Proto = asString(req["proto"])
		entry.RemoteIP = asString(req["remote_ip"])
		entry.ClientIP = asString(req["client_ip"])
		entry.UserAgent = headerValue(req["headers"], "User-Agent")
	}

	entry.Country = pickString(data, "country", "country_name", "country_name_zh", "country_name_zh-CN", "geoip2.country_names_zh-CN")
	entry.Province = pickString(data, "province", "province_name", "province_name_zh", "province_name_zh-CN", "geoip2.subdivisions_1_names_zh-CN")
	entry.City = pickString(data, "city", "city_name", "city_name_zh", "city_name_zh-CN", "geoip2.city_names_zh-CN")

	return entry, nil
}

func parseUnixTS(value any) (time.Time, bool) {
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

func asString(value any) string {
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

func asFloat(value any) float64 {
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

func headerValue(headers any, key string) string {
	m, ok := headers.(map[string]any)
	if !ok {
		return ""
	}
	val, ok := m[key]
	if !ok {
		return ""
	}
	switch v := val.(type) {
	case []any:
		if len(v) > 0 {
			return asString(v[0])
		}
	case []string:
		if len(v) > 0 {
			return v[0]
		}
	case string:
		return v
	}
	return ""
}

func pickString(data map[string]any, keys ...string) string {
	for _, key := range keys {
		if v, ok := data[key]; ok {
			if s := asString(v); s != "" {
				return s
			}
		}
	}
	return ""
}

func mustJSONRaw(line string) string {
	raw, err := json.Marshal(line)
	if err != nil {
		return "\"\""
	}
	return string(raw)
}
