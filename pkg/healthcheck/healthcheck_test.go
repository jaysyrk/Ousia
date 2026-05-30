package healthcheck

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jaysyrk/ousia/pkg/types"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Path != "/healthz" {
		t.Errorf("expected /healthz, got %s", cfg.Path)
	}
	if cfg.Interval != 10*time.Second {
		t.Errorf("expected 10s, got %v", cfg.Interval)
	}
}

func TestChecker_Add(t *testing.T) {
	checker := New(nil, DefaultConfig())
	checker.Add(&types.Endpoint{ID: "ep1"})
	if len(checker.endpoints) != 1 {
		t.Errorf("expected 1 endpoint, got %d", len(checker.endpoints))
	}
}

func TestChecker_Probe(t *testing.T) {
	srvOK := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srvOK.Close()

	srvErr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srvErr.Close()

	checker := New(nil, DefaultConfig())
	checker.cfg.Path = "/" // root path for test server

	// Test OK
	addrOK := strings.TrimPrefix(srvOK.URL, "http://")
	err := checker.probe(&types.Endpoint{Address: addrOK})
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// Test Err
	addrErr := strings.TrimPrefix(srvErr.URL, "http://")
	err = checker.probe(&types.Endpoint{Address: addrErr})
	if err == nil {
		t.Errorf("expected error, got nil")
	}

	// Test network error
	err = checker.probe(&types.Endpoint{Address: "127.0.0.1:0"})
	if err == nil {
		t.Errorf("expected network error, got nil")
	}
}

func TestChecker_Watch(t *testing.T) {
	var mu sync.Mutex
	var handlerStatus int = http.StatusOK

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		status := handlerStatus
		mu.Unlock()
		w.WriteHeader(status)
	}))
	defer srv.Close()

	addr := strings.TrimPrefix(srv.URL, "http://")

	cfg := Config{
		Path:             "/",
		Interval:         10 * time.Millisecond,
		Timeout:          50 * time.Millisecond,
		FailThreshold:    2,
		SuccessThreshold: 2,
	}

	ep := &types.Endpoint{
		ID:      "test-ep",
		Address: addr,
	}
	ep.Healthy.Store(true)

	checker := New([]*types.Endpoint{ep}, cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go checker.Start(ctx)

	mu.Lock()
	handlerStatus = http.StatusInternalServerError
	mu.Unlock()

	time.Sleep(50 * time.Millisecond)
	if ep.Healthy.Load() {
		t.Errorf("expected endpoint to be unhealthy")
	}

	mu.Lock()
	handlerStatus = http.StatusOK
	mu.Unlock()

	time.Sleep(50 * time.Millisecond)
	if !ep.Healthy.Load() {
		t.Errorf("expected endpoint to recover and be healthy")
	}
}
