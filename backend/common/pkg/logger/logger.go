// Package logger is a thin wrapper over log/slog so every service logs the
// same way. Init once at boot; the package functions use slog's default.
package logger

import (
	"context"
	"log/slog"
	"os"
)

// Init configures the process-wide logger. json selects the JSON handler
// (production); otherwise a text handler at debug level is used.
func Init(json bool) {
	var h slog.Handler
	if json {
		h = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	} else {
		h = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
	}
	slog.SetDefault(slog.New(h))
}

// Debug logs at debug level. args are slog key/value pairs.
func Debug(ctx context.Context, msg string, args ...any) { slog.DebugContext(ctx, msg, args...) }

// Info logs at info level.
func Info(ctx context.Context, msg string, args ...any) { slog.InfoContext(ctx, msg, args...) }

// Warn logs at warn level.
func Warn(ctx context.Context, msg string, args ...any) { slog.WarnContext(ctx, msg, args...) }

// Error logs at error level with err attached.
func Error(ctx context.Context, msg string, err error, args ...any) {
	if err != nil {
		args = append([]any{"err", err}, args...)
	}
	slog.ErrorContext(ctx, msg, args...)
}

// Fatal logs at error level and exits the process with status 1.
func Fatal(ctx context.Context, msg string, err error, args ...any) {
	Error(ctx, msg, err, args...)
	os.Exit(1)
}
