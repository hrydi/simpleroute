package simpleroute

import (
	"log"
)

// LogLevel represents the minimum level a log message must have to be emitted.
type LogLevel int

const (
	LogLevelError LogLevel = iota + 1
	LogLevelWarn
	LogLevelInfo
	LogLevelDebug
)

// Logger is the interface for leveled logging in the router.
// Implementations should respect the receiver's own level filtering,
// or use the LogLevel from RouterConfig for filtering.
type Logger interface {
	// Errorf logs a message at ERROR level.
	Errorf(format string, args ...any)
	// Warnf logs a message at WARN level.
	Warnf(format string, args ...any)
	// Infof logs a message at INFO level.
	Infof(format string, args ...any)
	// Debugf logs a message at DEBUG level.
	Debugf(format string, args ...any)
}

type defaultLogger struct {
	level LogLevel
}

func (l *defaultLogger) Errorf(format string, args ...any) {
	if l.level >= LogLevelError {
		log.Printf("[simpleroute] [ERROR] "+format, args...)
	}
}

func (l *defaultLogger) Warnf(format string, args ...any) {
	if l.level >= LogLevelWarn {
		log.Printf("[simpleroute] [WARN]  "+format, args...)
	}
}

func (l *defaultLogger) Infof(format string, args ...any) {
	if l.level >= LogLevelInfo {
		log.Printf("[simpleroute] [INFO]  "+format, args...)
	}
}

func (l *defaultLogger) Debugf(format string, args ...any) {
	if l.level >= LogLevelDebug {
		log.Printf("[simpleroute] [DEBUG] "+format, args...)
	}
}

func resolveLogger(config RouterConfig) Logger {
	if config.Logger != nil {
		return config.Logger
	}
	level := config.LogLevel
	if level < LogLevelError {
		level = LogLevelInfo
	}
	return &defaultLogger{level: level}
}
