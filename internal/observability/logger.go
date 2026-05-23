package observability

import (
	"log/slog"
	"os"
)

var Logger *slog.Logger

func InitLogger() {
	Logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(Logger)
}

func RequestLog(traceID, method, path, host, upstream string, statusCode int, durationMs float64) {
	Logger.Info("request",
		"trace_id", traceID,
		"method", method,
		"path", path,
		"host", host,
		"upstream", upstream,
		"status", statusCode,
		"duration_ms", durationMs,
	)
}
