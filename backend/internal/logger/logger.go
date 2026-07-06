package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
)

func Init(level, format, env string) {
	handler := newHandler(os.Stdout, level, format, env)
	slog.SetDefault(slog.New(handler))
}

func L() *slog.Logger {
	return slog.Default()
}

func Info(ctx context.Context, msg string, args ...any) {
	slog.InfoContext(ctx, msg, args...)
}

func Warn(ctx context.Context, msg string, args ...any) {
	slog.WarnContext(ctx, msg, args...)
}

func Error(ctx context.Context, msg string, args ...any) {
	slog.ErrorContext(ctx, msg, args...)
}

func newHandler(w io.Writer, level, format, env string) slog.Handler {
	opts := &slog.HandlerOptions{
		Level:     parseLevel(level),
		AddSource: strings.EqualFold(env, "dev") || strings.EqualFold(env, "local"),
	}
	if strings.EqualFold(format, "text") {
		return slog.NewTextHandler(w, opts)
	}
	return slog.NewJSONHandler(w, opts)
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
