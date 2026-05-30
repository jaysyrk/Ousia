package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestSidecarRegistrar(t *testing.T) {
	var registerHits atomic.Int32
	var hbHits atomic.Int32
	var deregisterHits atomic.Int32

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/mesh/register":
			registerHits.Add(1)
			w.WriteHeader(http.StatusCreated)
		case "/api/mesh/heartbeat":
			hbHits.Add(1)
			w.WriteHeader(http.StatusOK)
		case "/api/mesh/deregister":
			deregisterHits.Add(1)
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	r := NewSidecarRegistrar(ts.URL, "svc-a", "inst-1", "127.0.0.1", 8080, 50*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())

	go r.Start(ctx)

	time.Sleep(120 * time.Millisecond)
	cancel()
	time.Sleep(50 * time.Millisecond)

	if registerHits.Load() != 1 {
		t.Errorf("expected 1 register hit, got %d", registerHits.Load())
	}
	if hbHits.Load() < 1 {
		t.Errorf("expected at least 1 heartbeat hit, got %d", hbHits.Load())
	}
	if deregisterHits.Load() != 1 {
		t.Errorf("expected 1 deregister hit, got %d", deregisterHits.Load())
	}
}

func TestSidecarDiscovery(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/upstreams" {
			w.WriteHeader(http.StatusOK)
			payload := struct {
				Upstreams []upstreamResponse `json:"upstreams"`
			}{
				Upstreams: []upstreamResponse{
					{
						Name: "svc-a",
						Endpoints: []endpointResponse{
							{ID: "e1", Address: "10.0.0.1", Healthy: true},
							{ID: "e2", Address: "10.0.0.2", Healthy: false},
						},
					},
				},
			}
			json.NewEncoder(w).Encode(payload)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	mapper := NewServiceMapper()
	d := NewSidecarDiscovery(ts.URL, mapper, 50*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	go d.Start(ctx)

	time.Sleep(80 * time.Millisecond) // allow fetch
	cancel()

	addr, ok := mapper.Next("svc-a")
	if !ok {
		t.Fatalf("expected to find svc-a")
	}
	if addr != "10.0.0.1" {
		t.Errorf("expected 10.0.0.1, got %s", addr)
	}

	_, ok = mapper.Next("nonexistent")
	if ok {
		t.Errorf("expected false for nonexistent")
	}
}

func TestInboundProxy(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("local"))
	}))
	defer ts.Close()

	var port int
	fmt.Sscanf(ts.URL[strings.LastIndex(ts.URL, ":")+1:], "%d", &port)

	p := NewInboundProxy(port, "svc-a")
	go p.Start("127.0.0.1:29999")
	time.Sleep(100 * time.Millisecond)

	resp, err := http.Get("http://127.0.0.1:29999/test")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	resp2, _ := http.Get("http://127.0.0.1:29999/stats")
	if resp2.StatusCode != http.StatusOK {
		t.Errorf("expected 200 for stats, got %d", resp2.StatusCode)
	}
}

func TestOutboundProxy(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	mapper := NewServiceMapper()
	mapper.Update("svc-b", []string{ts.URL[7:]})

	p := NewOutboundProxy(mapper, "svc-a")
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Ousia-Service", "svc-b")

	w := httptest.NewRecorder()
	p.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.Header.Set("X-Ousia-Service", "unknown")
	w2 := httptest.NewRecorder()
	p.ServeHTTP(w2, req2)

	if w2.Code != http.StatusBadGateway {
		t.Errorf("expected 502, got %d", w2.Code)
	}
}

func TestRedirectToHTTPS(t *testing.T) {
	handler := redirectToHTTPS(":443")
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Host = "example.com"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusTemporaryRedirect {
		t.Errorf("expected 307, got %d", w.Code)
	}
	if loc := w.Header().Get("Location"); loc != "https://example.com/test" {
		t.Errorf("expected https location, got %s", loc)
	}
}

func TestSidecarRegistrar_Errors(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()
	r := NewSidecarRegistrar(ts.URL, "svc-a", "inst-1", "127.0.0.1", 8080, 50*time.Millisecond)
	r.register()
	r.heartbeat()
	r.deregister()
}

func TestSidecarDiscovery_Error(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	mapper := NewServiceMapper()
	d := NewSidecarDiscovery(ts.URL, mapper, 50*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	go d.Start(ctx)

	time.Sleep(80 * time.Millisecond)
	cancel()
	
	_, ok := mapper.Next("svc-a")
	if ok {
		t.Errorf("expected false")
	}
}
