package logging

import (
	"encoding/json"
	"sync"
	"sync/atomic"
	"time"

	"logflux/model"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const (
	defaultDBWriterBuffer       = 2048
	defaultDBWriterBatchSize    = 200
	defaultDBWriterFlushTimeout = 500 * time.Millisecond
)

// DBWriter 直接写入 system_logs 表，避免从文件采集后端日志。
type DBWriter struct {
	db           *gorm.DB
	source       string
	ch           chan *model.SystemLog
	wg           sync.WaitGroup
	closeOnce    sync.Once
	batchSize    int
	flushTimeout time.Duration
	dropped      uint64
}

// NewDBWriter 创建 DBWriter（默认后台异步入库）。
func NewDBWriter(db *gorm.DB, source string) *DBWriter {
	if db == nil {
		return nil
	}
	return newDBWriterWithOptions(db, source, defaultDBWriterBuffer, defaultDBWriterBatchSize, defaultDBWriterFlushTimeout)
}

func newDBWriterWithOptions(db *gorm.DB, source string, buffer int, batchSize int, flushTimeout time.Duration) *DBWriter {
	if buffer <= 0 {
		buffer = defaultDBWriterBuffer
	}
	if batchSize <= 0 {
		batchSize = defaultDBWriterBatchSize
	}
	if flushTimeout <= 0 {
		flushTimeout = defaultDBWriterFlushTimeout
	}

	writer := &DBWriter{
		db:           db.Session(&gorm.Session{Logger: logger.Discard, SkipDefaultTransaction: true}),
		source:       source,
		ch:           make(chan *model.SystemLog, buffer),
		batchSize:    batchSize,
		flushTimeout: flushTimeout,
	}
	writer.start()
	return writer
}

func (w *DBWriter) Alert(v any) {
	w.enqueue("alert", v)
}

func (w *DBWriter) Close() error {
	if w == nil {
		return nil
	}
	w.closeOnce.Do(func() {
		close(w.ch)
	})
	w.wg.Wait()
	return nil
}

func (w *DBWriter) Debug(v any, fields ...logx.LogField) {
	w.enqueue("debug", v, fields...)
}

func (w *DBWriter) Error(v any, fields ...logx.LogField) {
	w.enqueue("error", v, fields...)
}

func (w *DBWriter) Info(v any, fields ...logx.LogField) {
	w.enqueue("info", v, fields...)
}

func (w *DBWriter) Severe(v any) {
	w.enqueue("fatal", v)
}

func (w *DBWriter) Slow(v any, fields ...logx.LogField) {
	w.enqueue("slow", v, fields...)
}

func (w *DBWriter) Stack(v any) {
	w.enqueue("error", v)
}

func (w *DBWriter) Stat(v any, fields ...logx.LogField) {
	w.enqueue("stat", v, fields...)
}

func (w *DBWriter) enqueue(level string, v any, fields ...logx.LogField) {
	if w == nil {
		return
	}
	entry := buildSystemLog(level, v, fields...)
	entry.Source = w.source
	select {
	case w.ch <- entry:
	default:
		atomic.AddUint64(&w.dropped, 1)
	}
}

func (w *DBWriter) start() {
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		ticker := time.NewTicker(w.flushTimeout)
		defer ticker.Stop()

		batch := make([]*model.SystemLog, 0, w.batchSize)
		flush := func() {
			if len(batch) == 0 {
				return
			}
			_ = w.db.Create(&batch).Error
			batch = batch[:0]
		}

		for {
			select {
			case entry, ok := <-w.ch:
				if !ok {
					flush()
					return
				}
				batch = append(batch, entry)
				if len(batch) >= w.batchSize {
					flush()
				}
			case <-ticker.C:
				flush()
			}
		}
	}()
}

func buildSystemLog(level string, v any, fields ...logx.LogField) *model.SystemLog {
	entry := &model.SystemLog{
		LogTime:  time.Now(),
		Level:    level,
		Message:  formatValue(v),
		ExtraData: "{}",
	}

	extras := map[string]any{}
	for _, field := range fields {
		val := formatValue(field.Value)
		switch field.Key {
		case "caller":
			entry.Caller = val
		case "trace":
			entry.TraceID = val
		case "span":
			entry.SpanID = val
		default:
			extras[field.Key] = val
		}
	}

	if len(extras) > 0 {
		if payload, err := marshalJSON(extras); err == nil {
			entry.ExtraData = payload
		}
	}

	entry.RawLog = entry.Message
	return entry
}

func marshalJSON(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
