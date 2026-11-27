package ntgcalls

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
)

// LogLevel represents the severity level of a log message.
type LogLevel int

const (
	// LevelDebug enables debug level logging.
	LevelDebug LogLevel = iota
	// LevelInfo enables info level logging.
	LevelInfo
	// LevelWarn enables warning level logging.
	LevelWarn
	// LevelError enables error level logging.
	LevelError
	// LevelFatal enables fatal level logging.
	LevelFatal
)

// DefaultErrorLogPath is the file where ntgcalls/webrtc logs are saved.
const DefaultErrorLogPath = "error_log.txt"

var (
	sinkMu   sync.RWMutex
	sinkFile *os.File
)

// InitErrorLog opens (append mode) the given file and mirrors every log line
// printed by the ntgcalls and webrtc loggers into it, in addition to the terminal.
// Pass an empty path to use DefaultErrorLogPath.
func InitErrorLog(path string) error {
	if path == "" {
		path = DefaultErrorLogPath
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	sinkMu.Lock()
	old := sinkFile
	sinkFile = f
	sinkMu.Unlock()
	if old != nil {
		_ = old.Close()
	}
	return nil
}

// CloseErrorLog flushes and closes the error log file.
func CloseErrorLog() {
	sinkMu.Lock()
	defer sinkMu.Unlock()
	if sinkFile != nil {
		_ = sinkFile.Sync()
		_ = sinkFile.Close()
		sinkFile = nil
	}
}

// fileMinLevel is the lowest level written to error_log.txt.
// Default: everything that is printed. (Change to LevelWarn to keep only warnings/errors.)
var fileMinLevel = LevelDebug

// fileWriter writes to the error log file when it is configured.
type fileWriter struct{}

func (fileWriter) Write(p []byte) (int, error) {
	sinkMu.RLock()
	f := sinkFile
	sinkMu.RUnlock()
	if f == nil {
		return len(p), nil
	}
	return f.Write(p)
}

var _ io.Writer = fileWriter{}

// DebugEnabled reports whether NTG_DEBUG is set to a truthy value.
func DebugEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("NTG_DEBUG"))) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}

// SetDebug switches the ntgcalls logger to debug level and the webrtc logger
// to info level (very verbose). Turn off to return to the defaults.
func SetDebug(on bool) {
	if on {
		loggerNTGCalls.level = LevelDebug
		loggerWebRTC.level = LevelInfo
		fileMinLevel = LevelDebug
		return
	}
	loggerNTGCalls.level = LevelInfo
	loggerWebRTC.level = LevelFatal
	fileMinLevel = LevelDebug
}

// Logger provides structured logging for the ntgcalls package.
type Logger struct {
	name  string
	level LogLevel
	l     *log.Logger // terminal
	fl    *log.Logger // error_log.txt
}

// NewLogger creates a new Logger instance with the specified name and log level.
func NewLogger(name string, level LogLevel) *Logger {
	return &Logger{
		name:  name,
		level: level,
		l:     log.New(os.Stdout, fmt.Sprintf("[%s] ", name), log.LstdFlags),
		fl:    log.New(fileWriter{}, fmt.Sprintf("[%s] ", name), log.LstdFlags),
	}
}

func (lg *Logger) log(level LogLevel, tag string, msg string) {
	if level < lg.level {
		return
	}
	lg.l.Printf("[%s] %s", tag, msg)
	if level >= fileMinLevel {
		lg.fl.Printf("[%s] %s", tag, msg)
	}
}

// Debug logs a debug message.
func (lg *Logger) Debug(msg string) { lg.log(LevelDebug, "DEBUG", msg) }

// Info logs an informational message.
func (lg *Logger) Info(msg string) { lg.log(LevelInfo, "INFO", msg) }

// Warn logs a warning message.
func (lg *Logger) Warn(msg string) { lg.log(LevelWarn, "WARN", msg) }

// Error logs an error message.
func (lg *Logger) Error(msg string) { lg.log(LevelError, "ERROR", msg) }

// Fatal logs a fatal error message.
func (lg *Logger) Fatal(msg string) { lg.log(LevelFatal, "FATAL", msg) }
