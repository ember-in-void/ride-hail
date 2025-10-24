package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

type Level int

const (
	LevelInfo Level = iota
	LevelWarn
	LevelError
)

func parseLevel(s string) Level {
	switch s {
	case "warn", "warning":
		return LevelWarn
	case "error", "err":
		return LevelError
	default:
		return LevelInfo
	}
}

type Logger struct {
	service    string
	level      Level
	infoWriter io.Writer
	warnWriter io.Writer
	errWriter  io.Writer

	infoFile *os.File
	warnFile *os.File
	errFile  *os.File

	hostname string
	mu       sync.Mutex
}

// NewLogger создаёт JSON-логгер. dir — папка для логов (если пустая строка, "logs").
func NewLogger(service, levelStr, dir string) (*Logger, error) {
	if dir == "" {
		dir = "logs"
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create logs dir: %w", err)
	}

	infoPath := filepath.Join(dir, "info.log")
	warnPath := filepath.Join(dir, "warning.log")
	errPath := filepath.Join(dir, "error.log")

	infoF, err := os.OpenFile(infoPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o666)
	if err != nil {
		return nil, fmt.Errorf("open info log: %w", err)
	}
	warnF, err := os.OpenFile(warnPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o666)
	if err != nil {
		infoF.Close()
		return nil, fmt.Errorf("open warning log: %w", err)
	}
	errF, err := os.OpenFile(errPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o666)
	if err != nil {
		infoF.Close()
		warnF.Close()
		return nil, fmt.Errorf("open error log: %w", err)
	}

	hostname, _ := os.Hostname()

	l := &Logger{
		service:    service,
		level:      parseLevel(levelStr),
		infoWriter: io.MultiWriter(os.Stdout, infoF),
		warnWriter: io.MultiWriter(os.Stdout, warnF),
		errWriter:  io.MultiWriter(os.Stderr, errF),

		infoFile: infoF,
		warnFile: warnF,
		errFile:  errF,

		hostname: hostname,
	}

	return l, nil
}

func (l *Logger) Close() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.infoFile != nil {
		_ = l.infoFile.Close()
	}
	if l.warnFile != nil {
		_ = l.warnFile.Close()
	}
	if l.errFile != nil {
		_ = l.errFile.Close()
	}
}

func (l *Logger) Info(msg string, fields map[string]any) {
	l.log(LevelInfo, "info", msg, fields)
}

func (l *Logger) Infof(format string, args ...any) {
	l.Info(fmt.Sprintf(format, args...), nil)
}

func (l *Logger) Warn(msg string, fields map[string]any) {
	l.log(LevelWarn, "warn", msg, fields)
}

func (l *Logger) Error(msg string, fields map[string]any) {
	l.log(LevelError, "error", msg, fields)
}

// WithFields создаёт "дочерний" логгер с дополнительными полями,
// которые будут автоматически добавляться ко всем логам.
//
// Например:
//
//	reqLogger := log.WithFields(map[string]any{"request_id": "abc123"})
//	reqLogger.Info("Ride created", map[string]any{"ride_id": "ride42"})
//
// В результате в JSON появятся оба поля — "request_id" и "ride_id".
// Это удобно, когда нужно, чтобы все логи одного запроса имели общий контекст
// (например, request_id, user_id, ride_id и т.д.).
func (l *Logger) WithFields(base map[string]any) *entryLogger {
	return &entryLogger{parent: l, base: base}
}

type entryLogger struct {
	parent *Logger
	base   map[string]any
}

func (e *entryLogger) Info(msg string, fields map[string]any) {
	merged := mergeMaps(e.base, fields)
	e.parent.Info(msg, merged)
}

func (e *entryLogger) Warn(msg string, fields map[string]any) {
	merged := mergeMaps(e.base, fields)
	e.parent.Warn(msg, merged)
}

func (e *entryLogger) Error(msg string, fields map[string]any) {
	merged := mergeMaps(e.base, fields)
	e.parent.Error(msg, merged)
}

func mergeMaps(a, b map[string]any) map[string]any {
	if a == nil && b == nil {
		return nil
	}
	out := make(map[string]any, len(a)+len(b))
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		out[k] = v
	}
	return out
}

func (l *Logger) log(level Level, levelStr, msg string, fields map[string]any) {
	if level < l.level {
		return
	}

	entry := make(map[string]any, 8)
	entry["timestamp"] = time.Now().Local().Format("2006-01-02 15:04:05")
	entry["level"] = levelStr
	entry["service"] = l.service
	entry["message"] = msg
	entry["hostname"] = l.hostname

	// caller
	_, file, line, ok := runtime.Caller(2) // 2 — чтобы захватить внешний вызов
	if ok {
		entry["caller"] = fmt.Sprintf("%s:%d", file, line)
	} else {
		entry["caller"] = "unknown"
	}

	// merge user fields (request_id, ride_id, etc)
	for k, v := range fields {
		entry[k] = v
	}

	b, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		// fallback: plain text to errWriter
		l.mu.Lock()
		defer l.mu.Unlock()
		fmt.Fprintf(l.errWriter, `{"timestamp":"%s","level":"error","service":"%s","message":"failed to marshal log: %v"}`+"\n",
			time.Now().UTC().Format(time.Now().Format("2006-01-02 15:04:05")), l.service, err)
		return
	}

	// запись под мьютекс, чтобы строки логов не перемешивались
	l.mu.Lock()
	defer l.mu.Unlock()

	switch level {
	case LevelInfo:
		_, _ = l.infoWriter.Write(append(b, '\n'))
	case LevelWarn:
		_, _ = l.warnWriter.Write(append(b, '\n'))
	case LevelError:
		_, _ = l.errWriter.Write(append(b, '\n'))
	default:
		_, _ = l.infoWriter.Write(append(b, '\n'))
	}
}
