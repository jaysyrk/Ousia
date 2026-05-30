package observability

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestMiddleware(t *testing.T) {
	InitLogger()
	// reset stats
	atomic.StoreInt64(&currentStats.TotalRequests, 0)
	
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})

	handler := Middleware("test-upstream", nextHandler)

	req := httptest.NewRequest("GET", "http://example.com/foo", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", w.Code)
	}

	traceID := w.Header().Get("X-Trace-Id")
	if traceID == "" {
		t.Error("expected X-Trace-Id header in response")
	}

	if req.Header.Get("X-Trace-Id") != traceID {
		t.Error("expected X-Trace-Id header in request")
	}

	if atomic.LoadInt64(&currentStats.TotalRequests) != 1 {
		t.Errorf("expected total requests = 1")
	}
}
