package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"time"
)

// Level per spec: DEBUG, INFO, ERROR
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

func ParseLevel(s string) Level {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "DEBUG":
		return LevelDebug
	case "WARN":
		return LevelWarn
	case "ERROR":
		return LevelError
	default:
		return LevelInfo
	}
}

func levelString(l Level) string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "INFO"
	}
}

// ErrObj for error logs per spec
type ErrObj struct {
	Msg   string `json:"msg"`
	Stack string `json:"stack,omitempty"`
}

// Entry strictly follows the required schema
type Entry struct {
	Timestamp  string         `json:"timestamp"`            // ISO 8601 (UTC)
	Level      string         `json:"level"`                // INFO | DEBUG | WARN | ERROR
	Service    string         `json:"service"`              // e.g., ride-service
	Action     string         `json:"action"`               // event name, e.g., ride_requested
	Message    string         `json:"message"`              // human-readable
	Hostname   string         `json:"hostname"`             // container/host
	RequestID  string         `json:"request_id,omitempty"` // correlation id
	RideID     string         `json:"ride_id,omitempty"`    // when applicable
	Error      *ErrObj        `json:"error,omitempty"`      // only for ERROR
	Additional map[string]any `json:"additional,omitempty"` // optional extras
}

type Logger struct {
	service  string
	minLevel Level
	hostname string
	pretty   bool // если true, используем json.MarshalIndent

	outWriter io.Writer // stdout или MultiWriter
	errWriter io.Writer // stderr или MultiWriter для ошибок
	mu        sync.Mutex

	// optional dev file writers
	infoFile io.Closer
	errFile  io.Closer
}

// NewLogger stdout-only (recommended for prod)
func NewLogger(service string) *Logger {
	h, _ := os.Hostname()
	pretty := strings.ToLower(os.Getenv("LOG_PRETTY")) == "true"

	l := &Logger{
		service:   service,
		minLevel:  LevelInfo,
		hostname:  h,
		pretty:    pretty,
		outWriter: os.Stdout,
		errWriter: os.Stderr,
	}
	return l
}

// NewLoggerWithOptions supports minLevel and optional fileDir (dev).
// If fileDir != "", logs will also be duplicated into files (info.log, error.log).
func NewLoggerWithOptions(service, minLevelStr, fileDir string) (*Logger, error) {
	h, _ := os.Hostname()
	min := ParseLevel(minLevelStr)
	pretty := strings.ToLower(os.Getenv("LOG_PRETTY")) == "true"

	var outWriter io.Writer = os.Stdout
	var errWriter io.Writer = os.Stderr
	var infoCloser io.Closer
	var errCloser io.Closer

	if fileDir != "" {
		if err := os.MkdirAll(fileDir, 0o755); err != nil {
			return nil, fmt.Errorf("create logs dir: %w", err)
		}
		infoPath := filepath.Join(fileDir, "info.log")
		errPath := filepath.Join(fileDir, "error.log")

		infoF, err := os.OpenFile(infoPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o666)
		if err != nil {
			return nil, fmt.Errorf("open info log: %w", err)
		}
		errF, err := os.OpenFile(errPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o666)
		if err != nil {
			_ = infoF.Close()
			return nil, fmt.Errorf("open error log: %w", err)
		}

		outWriter = io.MultiWriter(os.Stdout, infoF)
		errWriter = io.MultiWriter(os.Stderr, errF)
		infoCloser, errCloser = infoF, errF

		l := &Logger{
			service:   service,
			minLevel:  min,
			hostname:  h,
			pretty:    pretty,
			outWriter: outWriter,
			errWriter: errWriter,
			infoFile:  infoCloser,
			errFile:   errCloser,
		}
		return l, nil
	}

	return &Logger{
		service:   service,
		minLevel:  min,
		hostname:  h,
		pretty:    pretty,
		outWriter: os.Stdout,
		errWriter: os.Stderr,
	}, nil
}

func (l *Logger) Close() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.infoFile != nil {
		_ = l.infoFile.Close()
	}
	if l.errFile != nil {
		_ = l.errFile.Close()
	}
}

func (l *Logger) Debug(e Entry) { l.log(LevelDebug, e, nil) }
func (l *Logger) Info(e Entry)  { l.log(LevelInfo, e, nil) }
func (l *Logger) Warn(e Entry)  { l.log(LevelWarn, e, nil) }
func (l *Logger) Error(e Entry) { l.log(LevelError, e, nil) }
func (l *Logger) Fatal(e Entry) {
	// include stack automatically for fatal
	if e.Error == nil {
		e.Error = &ErrObj{Msg: e.Message, Stack: string(debug.Stack())}
	} else if e.Error.Stack == "" {
		e.Error.Stack = string(debug.Stack())
	}
	l.log(LevelError, e, nil)
	os.Exit(1)
}

// WithFields returns a shallow "context" logger that auto-merges Additional fields.
func (l *Logger) WithFields(base map[string]any) *ContextLogger {
	return &ContextLogger{parent: l, base: base}
}

// WithContext is a helper to attach request_id and ride_id.
func (l *Logger) WithContext(requestID, rideID string) *ContextLogger {
	base := map[string]any{}
	if requestID != "" {
		base["request_id"] = requestID
	}
	if rideID != "" {
		base["ride_id"] = rideID
	}
	return &ContextLogger{parent: l, base: base}
}

type ContextLogger struct {
	parent *Logger
	base   map[string]any
}

func (c *ContextLogger) Debug(e Entry) { c.parent.log(LevelDebug, e, c.base) }
func (c *ContextLogger) Info(e Entry)  { c.parent.log(LevelInfo, e, c.base) }
func (c *ContextLogger) Warn(e Entry)  { c.parent.log(LevelWarn, e, c.base) }
func (c *ContextLogger) Error(e Entry) { c.parent.log(LevelError, e, c.base) }
func (c *ContextLogger) Fatal(e Entry) { c.parent.Fatal(mergeEntry(e, c.base)) }

func (l *Logger) log(level Level, e Entry, base map[string]any) {
	if level < l.minLevel {
		return
	}

	// fill required fields
	if e.Timestamp == "" {
		e.Timestamp = time.Now().UTC().Format(time.RFC3339Nano)
	}
	if e.Level == "" {
		e.Level = levelString(level)
	}
	if e.Service == "" {
		e.Service = l.service
	}
	if e.Hostname == "" {
		e.Hostname = l.hostname
	}
	// ensure action/message keys exist even if empty
	if e.Action == "" {
		e.Action = ""
	}
	if e.Message == "" {
		e.Message = ""
	}
	// ensure request_id/ride_id keys exist (empty string okay)
	if e.RequestID == "" {
		e.RequestID = toString(get(base, "request_id"))
	}
	if e.RideID == "" {
		e.RideID = toString(get(base, "ride_id"))
	}

	// merge Additional
	if base != nil {
		if e.Additional == nil {
			e.Additional = map[string]any{}
		}
		for k, v := range base {
			// do not overwrite required fields already set in Entry
			switch k {
			case "timestamp", "level", "service", "action", "message", "hostname", "request_id", "ride_id":
				continue
			default:
				e.Additional[k] = v
			}
		}
	}

	// caller enrichment (optional extra)
	if e.Additional == nil {
		e.Additional = make(map[string]any)
	}
	if _, ok := e.Additional["caller"]; !ok {
		if pc, file, line, ok := runtime.Caller(3); ok {
			fn := runtime.FuncForPC(pc)
			e.Additional["caller"] = fmt.Sprintf("%s:%d (%s)", file, line, funcName(fn))
		}
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// Выбираем writer в зависимости от уровня
	writer := l.outWriter
	if level == LevelError {
		writer = l.errWriter
	}

	// Сериализуем JSON
	var b []byte
	var err error

	if l.pretty {
		// Красивый вывод с отступами
		b, err = json.MarshalIndent(e, "", "  ")
	} else {
		// Компактный вывод (одна строка)
		b, err = json.Marshal(e)
	}

	if err != nil {
		// fallback: plain text to errWriter
		fmt.Fprintf(l.errWriter, `{"timestamp":"%s","level":"error","service":"%s","message":"failed to marshal log: %v"}`+"\n",
			time.Now().UTC().Format(time.RFC3339Nano), l.service, err)
		return
	}

	// Пишем лог
	_, _ = writer.Write(b)
	_, _ = writer.Write([]byte("\n"))

	// Дублируем ERROR в errFile если есть
	if level == LevelError && l.errFile != nil && l.errFile != l.infoFile {
		_, _ = l.errFile.(io.Writer).Write(b)
		_, _ = l.errFile.(io.Writer).Write([]byte("\n"))
	}
}

func funcName(fn *runtime.Func) string {
	if fn == nil {
		return "unknown"
	}
	return fn.Name()
}

func get(m map[string]any, k string) any {
	if m == nil {
		return ""
	}
	if v, ok := m[k]; ok {
		return v
	}
	return ""
}

func toString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	default:
		return ""
	}
}

func mergeEntry(e Entry, base map[string]any) Entry {
	if base == nil {
		return e
	}
	if e.Additional == nil {
		e.Additional = map[string]any{}
	}
	for k, v := range base {
		switch k {
		case "timestamp", "level", "service", "action", "message", "hostname", "request_id", "ride_id":
			continue
		default:
			e.Additional[k] = v
		}
	}
	if e.RequestID == "" {
		e.RequestID = toString(base["request_id"])
	}
	if e.RideID == "" {
		e.RideID = toString(base["ride_id"])
	}
	return e
}
