package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/zyanho/chameleon/pkg/plugin"
	"go.uber.org/zap"
)

// zapLogger adapts zap.Logger to plugin.Logger interface
type zapLogger struct {
	log *zap.Logger
}

func (l *zapLogger) Debug(msg string, args ...any) {
	l.log.Debug(msg, convertToZapFields(args...)...)
}

func (l *zapLogger) Info(msg string, args ...any) {
	l.log.Info(msg, convertToZapFields(args...)...)
}

func (l *zapLogger) Warn(msg string, args ...any) {
	l.log.Warn(msg, convertToZapFields(args...)...)
}

func (l *zapLogger) Error(msg string, args ...any) {
	l.log.Error(msg, convertToZapFields(args...)...)
}

// convertToZapFields converts interface arguments to zap fields
func convertToZapFields(args ...any) []zap.Field {
	if len(args) == 0 {
		return nil
	}

	fields := make([]zap.Field, 0, len(args)/2)
	for i := 0; i < len(args); i += 2 {
		if i+1 < len(args) {
			fields = append(fields, zap.Any(args[i].(string), args[i+1]))
		}
	}
	return fields
}

func main() {
	// Example 1: Using default logger
	config := plugin.DefaultConfig()
	manager1, _ := plugin.NewManager(context.Background(), config)
	defer manager1.Close()

	// Example 2: Using slog
	slogger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slogAdapter := &slogLogger{slogger}
	manager2, _ := plugin.NewManager(
		context.Background(),
		config,
		plugin.WithLogger(slogAdapter),
	)
	defer manager2.Close()

	// Example 3: Using zap
	zaplog, _ := zap.NewProduction()
	zapAdapter := &zapLogger{zaplog}
	manager3, _ := plugin.NewManager(
		context.Background(),
		config,
		plugin.WithLogger(zapAdapter),
	)
	defer manager3.Close()
}

// slog adapter
type slogLogger struct {
	log *slog.Logger
}

func (l *slogLogger) Debug(msg string, args ...any) {
	l.log.Debug(msg, args...)
}

func (l *slogLogger) Info(msg string, args ...any) {
	l.log.Info(msg, args...)
}

func (l *slogLogger) Warn(msg string, args ...any) {
	l.log.Warn(msg, args...)
}

func (l *slogLogger) Error(msg string, args ...any) {
	l.log.Error(msg, args...)
}
