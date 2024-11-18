package plugin

import (
	"log"
)

// Logger defines the interface for plugin logging
type Logger interface {
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
}

// DefaultLogger provides a basic implementation of the Logger interface
type DefaultLogger struct {
	level LogLevel
}

// NewDefaultLogger creates a default logger implementation
func NewDefaultLogger(level LogLevel) *DefaultLogger {
	return &DefaultLogger{level: level}
}

func (l *DefaultLogger) log(level LogLevel, msg string, args ...interface{}) {
	if level >= l.level {
		if len(args) > 0 {
			log.Printf("[%s] %s %v", levelToString(level), msg, args)
		} else {
			log.Printf("[%s] %s", levelToString(level), msg)
		}
	}
}

func (l *DefaultLogger) Debug(msg string, args ...interface{}) {
	l.log(LogLevelDebug, msg, args...)
}

func (l *DefaultLogger) Info(msg string, args ...interface{}) {
	l.log(LogLevelInfo, msg, args...)
}

func (l *DefaultLogger) Warn(msg string, args ...interface{}) {
	l.log(LogLevelWarn, msg, args...)
}

func (l *DefaultLogger) Error(msg string, args ...interface{}) {
	l.log(LogLevelError, msg, args...)
}

func levelToString(level LogLevel) string {
	switch level {
	case LogLevelDebug:
		return "DEBUG"
	case LogLevelInfo:
		return "INFO"
	case LogLevelWarn:
		return "WARN"
	case LogLevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}
