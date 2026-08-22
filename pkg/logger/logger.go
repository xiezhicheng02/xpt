package logger

import (
	"log/slog"
	"os"
)

func InitLogger(debug bool) {
	level := slog.LevelInfo
	if debug {
		level = slog.LevelDebug
	}
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	slog.SetDefault(slog.New(handler))
}

// WithComponent 返回带 component 字段的 logger，便于在多服务日志中区分来源。
func WithComponent(component string) *slog.Logger {
	return slog.Default().With("component", component)
}
