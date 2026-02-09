package ingest

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"logflux/model"

	"github.com/nxadm/tail"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type systemDirWatcher struct {
	stopCh   chan struct{}
	interval time.Duration
	source   string
}

type pendingEntry struct {
	entry    *model.SystemLog
	rawLines []string
	created  time.Time
}

// SystemIngestor 用于入库后端与 Caddy 后台运行日志
type SystemIngestor struct {
	db          *gorm.DB
	tails       map[string]*tail.Tail
	dirWatchers map[string]systemDirWatcher
	dirFiles    map[string]map[string]struct{}
	fileSource  map[string]string
	pending     map[string]*pendingEntry
	mu          sync.Mutex
}

func NewSystemIngestor(db *gorm.DB) *SystemIngestor {
	ing := &SystemIngestor{
		db:          db,
		tails:       make(map[string]*tail.Tail),
		dirWatchers: make(map[string]systemDirWatcher),
		dirFiles:    make(map[string]map[string]struct{}),
		fileSource:  make(map[string]string),
		pending:     make(map[string]*pendingEntry),
	}
	ing.startPendingFlush()
	return ing
}

func (i *SystemIngestor) StartWithInterval(filePath string, scanIntervalSec int, sourceType string) {
	filePath = strings.TrimSpace(filePath)
	if filePath == "" {
		return
	}
	filePath = filepath.Clean(filePath)

	if info, err := os.Stat(filePath); err == nil && info.IsDir() {
		i.startDir(filePath, scanIntervalSec, sourceType)
		return
	}

	i.startFile(filePath, sourceType)
}

func (i *SystemIngestor) startFile(filePath string, sourceType string) bool {
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
	i.fileSource[filePath] = normalizeSourceType(sourceType)
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
			i.mu.Lock()
			source := i.fileSource[path]
			i.mu.Unlock()
			if err := i.IngestLine(path, source, line.Text); err != nil {
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

func (i *SystemIngestor) resolveStartOffset(filePath string) int64 {
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

func (i *SystemIngestor) saveOffset(filePath string, offset int64) error {
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

func (i *SystemIngestor) startDir(dirPath string, scanIntervalSec int, sourceType string) {
	if scanIntervalSec <= 0 {
		scanIntervalSec = defaultScanIntervalSec
	}
	interval := time.Duration(scanIntervalSec) * time.Second

	var oldStopCh chan struct{}
	i.mu.Lock()
	if watcher, exists := i.dirWatchers[dirPath]; exists {
		if watcher.interval == interval && watcher.source == normalizeSourceType(sourceType) {
			i.mu.Unlock()
			return
		}
		oldStopCh = watcher.stopCh
	}
	stopCh := make(chan struct{})
	i.dirWatchers[dirPath] = systemDirWatcher{stopCh: stopCh, interval: interval, source: normalizeSourceType(sourceType)}
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

func (i *SystemIngestor) scanDir(dirPath string) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		logx.Errorf("Error reading dir: %v", err)
		return
	}

	i.mu.Lock()
	watcher, ok := i.dirWatchers[dirPath]
	i.mu.Unlock()
	if !ok {
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

		if i.startFile(filePath, watcher.source) {
			i.mu.Lock()
			if dirFiles, ok := i.dirFiles[dirPath]; ok {
				dirFiles[filePath] = struct{}{}
			}
			i.mu.Unlock()
		}
	}
}

func (i *SystemIngestor) Stop(filePath string) {
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

func (i *SystemIngestor) stopFile(filePath string) {
	i.mu.Lock()
	t, exists := i.tails[filePath]
	if exists {
		delete(i.tails, filePath)
		delete(i.fileSource, filePath)
		delete(i.pending, filePath)
	}
	i.mu.Unlock()

	if exists {
		t.Stop()
		t.Cleanup()
		logx.Infof("Stopped monitoring: %s", filePath)
	}
}

func (i *SystemIngestor) IngestLine(filePath string, sourceType string, line string) error {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil
	}

	if normalizeSourceType(sourceType) == "caddy_runtime" {
		if entry, err := parseCaddyRuntimeJSON(line); err == nil {
			entry.FilePath = filePath
			entry.Source = "caddy_runtime"
			entry.RawLog = line
			return i.save(entry)
		}
		// fallthrough to plain parse if json decode failed
	}

	if entry, ok := parsePlainMainLine(line); ok {
		entry.FilePath = filePath
		entry.Source = normalizeSourceType(sourceType)
		i.flushPending(filePath)
		i.mu.Lock()
		i.pending[filePath] = &pendingEntry{
			entry:    entry,
			rawLines: []string{line},
			created:  time.Now(),
		}
		i.mu.Unlock()
		return nil
	}

	if trace, span, extras, ok := parseTraceLine(line); ok {
		i.mu.Lock()
		pending := i.pending[filePath]
		i.mu.Unlock()
		if pending != nil {
			if trace != "" {
				pending.entry.TraceID = trace
			}
			if span != "" {
				pending.entry.SpanID = span
			}
			if len(extras) > 0 {
				mergeExtra(pending.entry, extras)
			}
			pending.rawLines = append(pending.rawLines, line)
			pending.entry.RawLog = strings.Join(pending.rawLines, "\n")
			i.mu.Lock()
			delete(i.pending, filePath)
			i.mu.Unlock()
			return i.save(pending.entry)
		}
		return nil
	}

	// 未识别的行：若有 pending，追加 raw 并立即落库，避免长期悬挂
	i.mu.Lock()
	pending := i.pending[filePath]
	if pending != nil {
		delete(i.pending, filePath)
	}
	i.mu.Unlock()
	if pending != nil {
		pending.rawLines = append(pending.rawLines, line)
		pending.entry.RawLog = strings.Join(pending.rawLines, "\n")
		return i.save(pending.entry)
	}

	return nil
}

func (i *SystemIngestor) save(entry *model.SystemLog) error {
	if entry.RawLog == "" {
		entry.RawLog = entry.Message
	}
	if entry.ExtraData == "" {
		entry.ExtraData = "{}"
	}
	return i.db.Create(entry).Error
}

func (i *SystemIngestor) flushPending(filePath string) {
	i.mu.Lock()
	pending := i.pending[filePath]
	if pending != nil {
		delete(i.pending, filePath)
	}
	i.mu.Unlock()
	if pending == nil {
		return
	}
	pending.entry.RawLog = strings.Join(pending.rawLines, "\n")
	if err := i.save(pending.entry); err != nil {
		logx.Errorf("Flush pending log failed: %v", err)
	}
}

func (i *SystemIngestor) startPendingFlush() {
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			now := time.Now()
			var stale []*pendingEntry
			var paths []string
			i.mu.Lock()
			for path, pending := range i.pending {
				if now.Sub(pending.created) >= 2*time.Second {
					stale = append(stale, pending)
					paths = append(paths, path)
				}
			}
			for _, path := range paths {
				delete(i.pending, path)
			}
			i.mu.Unlock()

			for _, pending := range stale {
				pending.entry.RawLog = strings.Join(pending.rawLines, "\n")
				if err := i.save(pending.entry); err != nil {
					logx.Errorf("Flush pending log failed: %v", err)
				}
			}
		}
	}()
}

func parseCaddyRuntimeJSON(line string) (*model.SystemLog, error) {
	decoder := json.NewDecoder(strings.NewReader(line))
	decoder.UseNumber()
	var data map[string]any
	if err := decoder.Decode(&data); err != nil {
		return nil, err
	}

	entry := &model.SystemLog{
		Source:    "caddy_runtime",
		ExtraData: "{}",
	}

	if ts, ok := parseUnixTS(data["ts"]); ok {
		entry.LogTime = ts
	} else {
		entry.LogTime = time.Now()
	}
	entry.Level = asString(data["level"])
	entry.Message = asString(data["msg"])

	extra := map[string]any{}
	for k, v := range data {
		if k == "ts" || k == "level" || k == "msg" {
			continue
		}
		extra[k] = v
	}
	if len(extra) > 0 {
		if b, err := json.Marshal(extra); err == nil {
			entry.ExtraData = string(b)
		}
	}

	return entry, nil
}

var (
	plainLineSep  = "\t"
	timeLayouts   = []string{"2006-01-02 15:04:05.000", "2006-01-02 15:04:05"}
	callerPattern = regexp.MustCompile(`.+\.go:\d+$`)
)

func parsePlainMainLine(line string) (*model.SystemLog, bool) {
	parts := strings.Split(line, plainLineSep)
	if len(parts) < 3 {
		return nil, false
	}

	ts := strings.TrimSpace(parts[0])
	level := strings.TrimSpace(parts[1])
	msg := strings.TrimSpace(parts[2])
	if ts == "" || level == "" {
		return nil, false
	}

	logTime, ok := parsePlainTime(ts)
	if !ok {
		return nil, false
	}

	entry := &model.SystemLog{
		LogTime: logTime,
		Level:   level,
		Message: msg,
	}

	extras := map[string]string{}
	caller := ""
	for _, part := range parts[3:] {
		token := strings.TrimSpace(part)
		if token == "" {
			continue
		}
		if key, val, ok := splitKV(token, "="); ok {
			extras[key] = val
			continue
		}
		if key, val, ok := splitKV(token, ":"); ok && (key == "trace" || key == "span" || key == "caller") {
			extras[key] = val
			continue
		}
		if caller == "" && looksLikeCaller(token) {
			caller = token
			continue
		}
		extras[token] = ""
	}

	if caller == "" {
		if v := extras["caller"]; v != "" {
			caller = v
			delete(extras, "caller")
		}
	}
	entry.Caller = caller
	if v := extras["trace"]; v != "" {
		entry.TraceID = v
		delete(extras, "trace")
	}
	if v := extras["span"]; v != "" {
		entry.SpanID = v
		delete(extras, "span")
	}

	if len(extras) > 0 {
		mergeExtra(entry, extras)
	}

	entry.RawLog = line
	return entry, true
}

func parseTraceLine(line string) (string, string, map[string]string, bool) {
	if !strings.Contains(line, "trace") && !strings.Contains(line, "span") {
		return "", "", nil, false
	}
	parts := strings.Split(line, plainLineSep)
	trace := ""
	span := ""
	extras := map[string]string{}
	for _, part := range parts {
		token := strings.TrimSpace(part)
		if token == "" {
			continue
		}
		if key, val, ok := splitKV(token, ":"); ok {
			switch key {
			case "trace":
				trace = val
			case "span":
				span = val
			default:
				extras[key] = val
			}
			continue
		}
		if key, val, ok := splitKV(token, "="); ok {
			switch key {
			case "trace":
				trace = val
			case "span":
				span = val
			default:
				extras[key] = val
			}
		}
	}
	if trace == "" && span == "" && len(extras) == 0 {
		return "", "", nil, false
	}
	return trace, span, extras, true
}

func splitKV(token string, sep string) (string, string, bool) {
	if !strings.Contains(token, sep) {
		return "", "", false
	}
	parts := strings.SplitN(token, sep, 2)
	if len(parts) != 2 {
		return "", "", false
	}
	key := strings.TrimSpace(parts[0])
	val := strings.TrimSpace(parts[1])
	if key == "" || val == "" {
		return "", "", false
	}
	return key, val, true
}

func parsePlainTime(ts string) (time.Time, bool) {
	for _, layout := range timeLayouts {
		if t, err := time.ParseInLocation(layout, ts, time.Local); err == nil {
			return t, true
		}
		if t, err := time.Parse(layout, ts); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

func looksLikeCaller(token string) bool {
	return callerPattern.MatchString(token)
}

func mergeExtra(entry *model.SystemLog, extras map[string]string) {
	payload := map[string]any{}
	if entry.ExtraData != "" && entry.ExtraData != "{}" {
		_ = json.Unmarshal([]byte(entry.ExtraData), &payload)
	}
	for k, v := range extras {
		if v == "" {
			payload[k] = true
		} else {
			payload[k] = v
		}
	}
	if len(payload) == 0 {
		entry.ExtraData = "{}"
		return
	}
	if b, err := json.Marshal(payload); err == nil {
		entry.ExtraData = string(b)
	}
}
