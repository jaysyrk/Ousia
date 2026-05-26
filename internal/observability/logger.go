package observability

import (
	"log/slog"
	"os"
	"sync/atomic"
	"time"
)

var Logger *slog.Logger

type Snapshot struct {
	TotalRequests int64
	TotalErrors   int64
	TotalDuration int64
}

var currentStats Snapshot

func InitLogger() {
	Logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(Logger)

	go startAsyncBatchFlusher()
}

func startAsyncBatchFlusher() {
	ticker := time.NewTicker(time.Second)
	for range ticker.C {
		// Read and reset atomics
		reqs := atomic.SwapInt64(&currentStats.TotalRequests, 0)
		errs := atomic.SwapInt64(&currentStats.TotalErrors, 0)
		dur := atomic.SwapInt64(&currentStats.TotalDuration, 0)

		if reqs > 0 {
			avgLatency := float64(dur) / float64(reqs)
			Logger.Info("1-Second Snapshot Flush",
				"requests_per_sec", reqs,
				"errors", errs,
				"avg_latency_ms", avgLatency,
			)
		}
	}
}

func RequestLog(traceID, method, path, host, upstream string, statusCode int, durationMs float64) {
	// Instead of blocking I/O on every request, silently update the atomics in RAM
	atomic.AddInt64(&currentStats.TotalRequests, 1)
	atomic.AddInt64(&currentStats.TotalDuration, int64(durationMs))
	if statusCode >= 400 {
		atomic.AddInt64(&currentStats.TotalErrors, 1)
	}
}
