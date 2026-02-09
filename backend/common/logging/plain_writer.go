package logging

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

const defaultTimeFormat = "2006-01-02 15:04:05"

// PlainConsoleWriter 使用固定格式输出日志，避免 JSON 控制台输出。
type PlainConsoleWriter struct {
	out        io.Writer
	timeFormat string
	mu         sync.Mutex
}

// NewPlainConsoleWriter 创建控制台日志 Writer。
func NewPlainConsoleWriter(out io.Writer, timeFormat string) logx.Writer {
	if out == nil {
		out = os.Stdout
	}
	if timeFormat == "" {
		timeFormat = defaultTimeFormat
	}
	return &PlainConsoleWriter{
		out:        out,
		timeFormat: timeFormat,
	}
}

func (w *PlainConsoleWriter) Alert(v any) {
	w.write("alert", v)
}

func (w *PlainConsoleWriter) Close() error {
	return nil
}

func (w *PlainConsoleWriter) Debug(v any, fields ...logx.LogField) {
	w.write("debug", v, fields...)
}

func (w *PlainConsoleWriter) Error(v any, fields ...logx.LogField) {
	w.write("error", v, fields...)
}

func (w *PlainConsoleWriter) Info(v any, fields ...logx.LogField) {
	w.write("info", v, fields...)
}

func (w *PlainConsoleWriter) Severe(v any) {
	w.write("fatal", v)
}

func (w *PlainConsoleWriter) Slow(v any, fields ...logx.LogField) {
	w.write("slow", v, fields...)
}

func (w *PlainConsoleWriter) Stack(v any) {
	w.write("error", v)
}

func (w *PlainConsoleWriter) Stat(v any, fields ...logx.LogField) {
	w.write("stat", v, fields...)
}

func (w *PlainConsoleWriter) write(level string, v any, fields ...logx.LogField) {
	msg := formatValue(v)
	timestamp := time.Now().Format(w.timeFormat)

	var caller string
	var traceID string
	var spanID string
	extras := make([]string, 0, len(fields))

	for _, field := range fields {
		value := formatValue(field.Value)
		switch field.Key {
		case "caller":
			caller = value
		case "trace":
			traceID = value
		case "span":
			spanID = value
		default:
			extras = append(extras, fmt.Sprintf("%s: %s", field.Key, value))
		}
	}

	line1Parts := []string{timestamp, level, msg}
	if caller != "" {
		line1Parts = append(line1Parts, caller)
	}
	line1 := strings.Join(line1Parts, "\t")

	lines := []string{line1}
	line2Parts := make([]string, 0, 2+len(extras))
	if traceID != "" {
		line2Parts = append(line2Parts, fmt.Sprintf("trace: %s", traceID))
	}
	if spanID != "" {
		line2Parts = append(line2Parts, fmt.Sprintf("span: %s", spanID))
	}
	if len(extras) > 0 {
		line2Parts = append(line2Parts, extras...)
	}
	if len(line2Parts) > 0 {
		lines = append(lines, strings.Join(line2Parts, "\t"))
	}

	payload := strings.Join(lines, "\n") + "\n"

	w.mu.Lock()
	defer w.mu.Unlock()
	_, _ = w.out.Write([]byte(payload))
}
