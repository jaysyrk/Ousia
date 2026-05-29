package gateway

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/jaysyrk/ousia/internal/balancer"
	"github.com/jaysyrk/ousia/internal/router"
	"github.com/jaysyrk/ousia/pkg/types"
)

type mockBalancer struct {
	ep        *types.Endpoint
	nextCalls int
	doneCalls int
	mu        sync.Mutex
}

func (m *mockBalancer) Next(key string) (*types.Endpoint, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nextCalls++
	return m.ep, nil
}

func (m *mockBalancer) Done(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.doneCalls++
}

func (m *mockBalancer) Add(ep *types.Endpoint)    {}
func (m *mockBalancer) Remove(id string) {}

func TestGateway_BalancerDoneCalled(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	addr := backend.URL[7:]
	
	ep := &types.Endpoint{ID: "test-ep", Address: addr, Healthy: true}
	mb := &mockBalancer{ep: ep}

	vh := &types.VirtualHost{
		Hostname: "*",
		Routes: []*types.Route{
			{
				ID: "test-route",
				Match: types.RouteMatch{PathPrefix: "/"},
				Action: types.RouteAction{UpstreamPool: "test-pool"},
			},
		},
	}
	rt := router.New([]*types.VirtualHost{vh})

	h := NewHandler(rt, map[string]balancer.Balancer{"test-pool": mb})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}

	mb.mu.Lock()
	defer mb.mu.Unlock()
	if mb.nextCalls != 1 {
		t.Errorf("expected Next() to be called 1 time, got %d", mb.nextCalls)
	}
	if mb.doneCalls != 1 {
		t.Errorf("expected Done() to be called 1 time, got %d", mb.doneCalls)
	}
}

func TestGateway_ResiliencyRetry(t *testing.T) {
	var attempts int
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))
	defer backend.Close()

	addr := backend.URL[7:]
	ep := &types.Endpoint{ID: "retry-ep", Address: addr, Healthy: true}
	mb := &mockBalancer{ep: ep}

	vh := &types.VirtualHost{
		Hostname: "*",
		Routes: []*types.Route{
			{
				ID: "retry-route",
				Match: types.RouteMatch{PathPrefix: "/"},
				Action: types.RouteAction{
					UpstreamPool: "test-pool",
					RetryCount:   3,
				},
			},
		},
	}
	rt := router.New([]*types.VirtualHost{vh})

	h := NewHandler(rt, map[string]balancer.Balancer{"test-pool": mb})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}
	if attempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}
}

func TestGateway_CircuitBreaker(t *testing.T) {
	var upstreamHits int
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamHits++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer backend.Close()

	addr := backend.URL[7:]
	ep := &types.Endpoint{ID: "cb-ep", Address: addr, Healthy: true}
	mb := &mockBalancer{ep: ep}

	vh := &types.VirtualHost{
		Hostname: "*",
		Routes: []*types.Route{
			{
				ID: "cb-route",
				Match: types.RouteMatch{PathPrefix: "/"},
				Action: types.RouteAction{UpstreamPool: "test-pool"},
			},
		},
	}
	rt := router.New([]*types.VirtualHost{vh})

	h := NewHandler(rt, map[string]balancer.Balancer{"test-pool": mb})

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		
		if i == 0 {
			if w.Code != http.StatusInternalServerError {
				t.Fatalf("expected 500 Internal Server Error on request 1, got %d", w.Code)
			}
		} else {
			if w.Code != http.StatusBadGateway {
				t.Fatalf("expected 502 Bad Gateway on request 2, got %d", w.Code)
			}
		}
	}

	if upstreamHits != 3 {
		t.Fatalf("expected upstream to be hit exactly 3 times before CB opens, got %d", upstreamHits)
	}
}
