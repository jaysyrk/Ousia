package observability

import (
	"sync/atomic"
	"testing"
)

func TestInitLogger(t *testing.T) {
	InitLogger()
	if Logger == nil {
		t.Fatal("expected Logger to be initialized")
	}
}

func TestRequestLog(t *testing.T) {
	// reset stats
	atomic.StoreInt64(&currentStats.TotalRequests, 0)
	atomic.StoreInt64(&currentStats.TotalErrors, 0)
	atomic.StoreInt64(&currentStats.TotalDuration, 0)

	RequestLog("trace-1", "GET", "/test", "localhost", "up1", 200, 15.5)
	
	if currentStats.TotalRequests != 1 {
		t.Errorf("expected 1 request, got %d", currentStats.TotalRequests)
	}
	if currentStats.TotalErrors != 0 {
		t.Errorf("expected 0 errors, got %d", currentStats.TotalErrors)
	}
	if currentStats.TotalDuration != 15 {
		t.Errorf("expected 15 duration, got %d", currentStats.TotalDuration)
	}

	RequestLog("trace-2", "GET", "/error", "localhost", "up1", 500, 20.0)
	
	if currentStats.TotalRequests != 2 {
		t.Errorf("expected 2 requests, got %d", currentStats.TotalRequests)
	}
	if currentStats.TotalErrors != 1 {
		t.Errorf("expected 1 error, got %d", currentStats.TotalErrors)
	}
	if currentStats.TotalDuration != 35 {
		t.Errorf("expected 35 duration, got %d", currentStats.TotalDuration)
	}
}
