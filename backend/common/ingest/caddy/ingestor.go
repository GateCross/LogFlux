package caddy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/netip"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"logflux/common/ingest/shared"
	"logflux/internal/utils/safego"
	caddymodel "logflux/model/caddy"
	ingestmodel "logflux/model/ingest"

	"github.com/nxadm/tail"
	"github.com/zeromicro/go-zero/core/logx"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Log Format:
// [{ts}] "{country_name}" "{province_name}" "{city_name}" "{request>host}" "{request>method} {request>uri} {request>proto}" {status} {size} "{request>headers>User-Agent>[0]}" "{request>remote_ip}" "{request>client_ip}"

var logRegex = regexp.MustCompile(`^\[(.*?)\] "(.*?)" "(.*?)" "(.*?)" "(.*?)" "(.*?) (.*?) (.*?)" (\d+) (\d+) "(.*?)" "(.*?)" "(.*?)"$`)

const caddyInternalSource = "caddy_internal"

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

func (i *CaddyIngestor) ParseLine(line string) (*caddymodel.CaddyLog, error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil, fmt.Errorf("日志行为空")
	}

	if strings.HasPrefix(line, "{") {
		if logEntry, err := i.parseJSONLine(line); err == nil {
			return logEntry, nil
		}
	}

	matches := logRegex.FindStringSubmatch(line)
	if len(matches) != 14 {
		return nil, fmt.Errorf("日志格式无效: %s", line)
	}

	logTime, err := i.parseTime(matches[1])
	if err != nil {
		logx.Errorf("解析时间失败: %v，原始值=%s", err, matches[1])
	}

	status, _ := strconv.Atoi(matches[9])
	size, _ := strconv.ParseInt(matches[10], 10, 64)

	return &caddymodel.CaddyLog{
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
	return time.Now(), fmt.Errorf("未知时间格式")
}

func (i *CaddyIngestor) Ingest(line string) error {
	return i.IngestWithPath("", line)
}

func (i *CaddyIngestor) IngestWithPath(filePath string, line string) error {
	logEntry, err := i.ParseLine(line)
	if err != nil {
		return err
	}
	if isInternalCaddyAccess(logEntry) {
		if err := i.saveInternalAccessLog(filePath, line, logEntry); err != nil {
			logx.Errorf("写入系统日志失败: %v", err)
			return err
		}
		return nil
	}
	if err := i.db.Create(logEntry).Error; err != nil {
		logx.Errorf("写入数据库失败: %v", err)
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
		logx.Errorf("监听文件失败: %v", err)
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

	logx.Infof("开始监听文件: %s", filePath)

	watchPath := filePath
	safego.New(context.Background(), "Caddy 日志文件监听").Go(func() {
		path := watchPath
		for line := range t.Lines {
			if line == nil {
				continue
			}
			if line.Err != nil {
				logx.Errorf("读取监听内容失败: %v", line.Err)
				continue
			}
			if err := i.IngestWithPath(path, line.Text); err != nil {
				// keep noisy errors in stdout for now
				logx.Errorf("日志入库失败: %v", err)
				continue
			}
			if err := i.saveOffset(path, line.SeekInfo.Offset); err != nil {
				logx.Errorf("保存日志采集游标失败: %v", err)
			}
		}
	})

	return true
}

func (i *CaddyIngestor) resolveStartOffset(filePath string) int64 {
	var cursor ingestmodel.LogIngestCursor
	if err := i.db.Where("file_path = ?", filePath).Take(&cursor).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			logx.Errorf("加载日志采集游标失败: %v", err)
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

	cursor := ingestmodel.LogIngestCursor{
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
		scanIntervalSec = shared.DefaultScanIntervalSec()
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

	safego.New(context.Background(), "Caddy 日志目录扫描").Go(func() {
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
	})
}

func (i *CaddyIngestor) scanDir(dirPath string) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		logx.Errorf("读取目录失败: %v", err)
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !shared.IsLogFileName(name) {
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
		logx.Infof("停止监听文件: %s", filePath)
	}
}

func (i *CaddyIngestor) parseJSONLine(line string) (*caddymodel.CaddyLog, error) {
	decoder := json.NewDecoder(strings.NewReader(line))
	decoder.UseNumber()

	var data map[string]any
	if err := decoder.Decode(&data); err != nil {
		return nil, err
	}

	// Coraza WAF 审计日志格式：顶层含 "transaction" 字段
	if _, ok := data["transaction"]; ok {
		return parseCorazaAuditJSON(line, data)
	}

	entry := &caddymodel.CaddyLog{
		RawLog:    line,
		ExtraData: "{}",
	}

	if ts, ok := shared.ParseUnixTS(data["ts"]); ok {
		entry.LogTime = ts
	} else {
		entry.LogTime = time.Now()
	}

	entry.Status = int(shared.AsFloat(data["status"]))
	entry.Size = int64(shared.AsFloat(data["size"]))

	if req, ok := data["request"].(map[string]any); ok {
		entry.Host = shared.AsString(req["host"])
		entry.Method = shared.AsString(req["method"])
		entry.Uri = shared.AsString(req["uri"])
		entry.Proto = shared.AsString(req["proto"])
		entry.RemoteIP = shared.AsString(req["remote_ip"])
		entry.ClientIP = shared.AsString(req["client_ip"])
		entry.UserAgent = headerValue(req["headers"], "User-Agent")
	}

	entry.Country = pickString(data, "country", "country_name", "country_name_zh", "country_name_zh-CN", "geoip2.country_names_zh-CN")
	entry.Province = pickString(data, "province", "province_name", "province_name_zh", "province_name_zh-CN", "geoip2.subdivisions_1_names_zh-CN")
	entry.City = pickString(data, "city", "city_name", "city_name_zh", "city_name_zh-CN", "geoip2.city_names_zh-CN")

	return entry, nil
}

// parseCorazaAuditJSON 解析 Coraza WAF 审计日志 JSON 格式。
// Coraza 格式: {transaction:{...}, messages:[...], request:{...}, response:{...}}
func parseCorazaAuditJSON(line string, data map[string]any) (*caddymodel.CaddyLog, error) {
	entry := &caddymodel.CaddyLog{
		RawLog:    line,
		ExtraData: "{}",
	}

	// 提取 transaction 字段
	tx, _ := data["transaction"].(map[string]any)
	if tx != nil {
		if ts, ok := shared.ParseUnixTS(tx["unix_timestamp"]); ok {
			entry.LogTime = ts
		}
		entry.ClientIP = shared.AsString(tx["client_ip"])
		entry.RemoteIP = shared.AsString(tx["host_ip"])
	}

	if entry.LogTime.IsZero() {
		entry.LogTime = time.Now()
	}

	// 提取 request 字段
	if req, ok := data["request"].(map[string]any); ok {
		entry.Method = shared.AsString(req["method"])
		entry.Uri = shared.AsString(req["uri"])
		if hv, ok := req["http_version"].(float64); ok {
			entry.Proto = fmt.Sprintf("HTTP/%.1f", hv)
		}
		entry.Host = headerValue(req["headers"], "Host")
		entry.UserAgent = headerValue(req["headers"], "User-Agent")
	}

	// 提取 response 字段
	if resp, ok := data["response"].(map[string]any); ok {
		entry.Status = int(shared.AsFloat(resp["status"]))
	}

	// 提取 WAF 命中信息存入 ExtraData
	extra := map[string]any{}
	if tx != nil {
		extra["transaction_id"] = shared.AsString(tx["id"])
	}
	if msgs, ok := data["messages"].([]any); ok && len(msgs) > 0 {
		rules := make([]map[string]any, 0, len(msgs))
		for _, msg := range msgs {
			if m, ok := msg.(map[string]any); ok {
				rule := map[string]any{}
				if d, ok := m["data"].(map[string]any); ok {
					rule["id"] = d["id"]
					rule["rev"] = d["rev"]
					rule["msg"] = d["msg"]
					rule["severity"] = d["severity"]
					rule["ver"] = d["ver"]
				}
				rule["message"] = shared.AsString(m["message"])
				rule["actionset"] = shared.AsString(m["actionset"])
				rules = append(rules, rule)
			}
		}
		if len(rules) > 0 {
			extra["matched_rules"] = rules
		}
	}
	extra["source"] = "waf"

	if b, err := json.Marshal(extra); err == nil {
		entry.ExtraData = string(b)
	}

	return entry, nil
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
			return shared.AsString(v[0])
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
			if s := shared.AsString(v); s != "" {
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

func (i *CaddyIngestor) saveInternalAccessLog(filePath string, rawLine string, entry *caddymodel.CaddyLog) error {
	logTime := entry.LogTime
	if logTime.IsZero() {
		logTime = time.Now()
	}

	extra := map[string]any{
		"host":      entry.Host,
		"method":    entry.Method,
		"uri":       entry.Uri,
		"proto":     entry.Proto,
		"status":    entry.Status,
		"size":      entry.Size,
		"userAgent": entry.UserAgent,
		"remoteIP":  entry.RemoteIP,
		"clientIP":  entry.ClientIP,
		"country":   entry.Country,
		"province":  entry.Province,
		"city":      entry.City,
	}
	extraData := "{}"
	if data, err := json.Marshal(extra); err == nil {
		extraData = string(data)
	}

	systemLog := &ingestmodel.SystemLog{
		LogTime:   logTime,
		Level:     caddyAccessLevel(entry.Status),
		Message:   fmt.Sprintf("Caddy 本机访问 %s %s %d host=%s remote=%s client=%s", entry.Method, entry.Uri, entry.Status, entry.Host, entry.RemoteIP, entry.ClientIP),
		Source:    caddyInternalSource,
		FilePath:  strings.TrimSpace(filePath),
		RawLog:    rawLine,
		ExtraData: extraData,
	}
	return i.db.Create(systemLog).Error
}

// isInternalCaddyAccess 将本机和私有地址访问从代理访问日志中剥离。
func isInternalCaddyAccess(entry *caddymodel.CaddyLog) bool {
	if entry == nil {
		return false
	}
	if isPrivateAccessHost(entry.Host) {
		return true
	}

	return isPrivateAddressLiteral(entry.ClientIP)
}

func isPrivateAccessHost(value string) bool {
	host := normalizeAddressToken(value)
	if host == "" {
		return false
	}

	normalized := strings.ToLower(host)
	if normalized == "localhost" || strings.HasSuffix(normalized, ".localhost") || normalized == "localhost.localdomain" {
		return true
	}
	return isPrivateAddressLiteral(host)
}

func isPrivateAddressLiteral(value string) bool {
	host := normalizeAddressToken(value)
	if host == "" {
		return false
	}
	addr, err := netip.ParseAddr(host)
	if err != nil {
		return false
	}
	addr = addr.Unmap()
	return addr.IsLoopback() || addr.IsPrivate() || addr.IsLinkLocalUnicast() || addr.IsLinkLocalMulticast() || addr.IsUnspecified()
}

func normalizeAddressToken(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if idx := strings.Index(value, ","); idx >= 0 {
		value = strings.TrimSpace(value[:idx])
	}
	if fields := strings.Fields(value); len(fields) > 0 {
		value = fields[0]
	}
	value = strings.Trim(value, `"'`)

	if host, _, err := net.SplitHostPort(value); err == nil {
		return strings.Trim(host, "[]")
	}
	if strings.HasPrefix(value, "[") {
		if idx := strings.Index(value, "]"); idx > 0 {
			return strings.Trim(value[1:idx], "[]")
		}
	}
	if strings.Count(value, ":") == 1 {
		host, port, ok := strings.Cut(value, ":")
		if ok && host != "" && port != "" {
			if _, err := strconv.Atoi(port); err == nil {
				return strings.Trim(host, "[]")
			}
		}
	}
	return strings.Trim(value, "[]")
}

func caddyAccessLevel(status int) string {
	if status >= 500 {
		return "error"
	}
	if status >= 400 {
		return "warn"
	}
	return "info"
}
