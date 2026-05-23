package observability

import (
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type statusRecorder struct {
	http.ResponseWriter
	status	int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func Middleware(upstream string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {

		traceID := req.Header.Get("X-Trace-Id")
		if traceID == "" {
			traceID = uuid.New().String()
		}

		start := time.Now()

		w.Header().Set("X-Trace-Id", traceID)
		req.Header.Set("X-Trace-Id", traceID)

		ActiveConnections.Inc()
		defer ActiveConnections.Dec()

		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(rec, req)

		durationMs := float64(time.Since(start).Milliseconds())
		status := fmt.Sprintf("%d", rec.status)

		RequestsTotal.WithLabelValues(req.Method, req.Host, upstream, status).Inc()
		RequestDuration.WithLabelValues(req.Method, req.Host, upstream).Observe(durationMs)
		RequestLog(traceID, req.Method, req.URL.Path, req.Host, upstream, rec.status, durationMs)
	})
}
