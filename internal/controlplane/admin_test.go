package controlplane

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jaysyrk/ousia/internal/balancer"
	"github.com/jaysyrk/ousia/internal/router"
	"github.com/jaysyrk/ousia/pkg/config"
)

func TestAdminAPI(t *testing.T) {
	r := router.New(nil)
	balancers := make(map[string]balancer.Balancer)
	store := NewStore(&config.OusiaConfig{
		VirtualHosts: []config.VirtualHostConfig{
			{Hostname: "example.com", Routes: []config.RouteConfig{{ID: "r1", Action: config.ActionConfig{Upstream: "u1"}}}},
		},
		Upstreams: []config.UpstreamConfig{
			{Name: "u1", Algorithm: "round-robin", Endpoints: []config.EndpointConfig{{ID: "e1", Address: "127.0.0.1:80"}}},
		},
	})
	mesh := NewMeshRegistry(time.Minute)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	mesh.StartCleanup(ctx)

	updater := BuildUpdateFunc(r, balancers)
	updater(store.Get())

	api := NewAdminAPI(r, balancers, store, mesh)
	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	tests := []struct {
		method string
		path   string
		body   string
		code   int
	}{
		{"GET", "/api/routes", "", 200},
		{"POST", "/api/routes", `{"id":"r2","virtual_host":"example.com","upstream":"u1"}`, 201},
		{"GET", "/api/routes/r2", "", 200},
		{"GET", "/api/routes/invalid", "", 404},
		{"POST", "/api/routes", `{"id":"","virtual_host":"","upstream":""}`, 400},
		{"POST", "/api/routes", `invalid_json`, 400},
		{"PUT", "/api/routes/r2", `{"virtual_host":"example.com","upstream":"u1"}`, 200},
		{"PUT", "/api/routes/r2", `{"virtual_host":"","upstream":""}`, 400},
		{"PUT", "/api/routes/r2", `invalid_json`, 400},
		{"DELETE", "/api/routes/r2", "", 200},
		{"DELETE", "/api/routes/invalid", "", 404},
		{"GET", "/api/upstreams", "", 200},
		{"POST", "/api/upstreams/u1/endpoints", `{"id":"e2","address":"127.0.0.2:80"}`, 201},
		{"POST", "/api/upstreams/u1/endpoints", `{"id":"","address":""}`, 400},
		{"POST", "/api/upstreams/invalid/endpoints", `{"id":"e99","address":"localhost"}`, 404},
		{"DELETE", "/api/upstreams/u1/endpoints/e2", "", 200},
		{"DELETE", "/api/upstreams/invalid/endpoints/e1", "", 404},
		{"POST", "/api/mesh/register", `{"service_id":"s1","instance_id":"i1","address":"localhost","port":80}`, 201},
		{"POST", "/api/mesh/register", `invalid_json`, 400},
		{"POST", "/api/mesh/register", `{"service_id":"","instance_id":""}`, 400},
		{"POST", "/api/mesh/heartbeat", `{"instance_id":"i1"}`, 200},
		{"POST", "/api/mesh/heartbeat", `invalid_json`, 400},
		{"POST", "/api/mesh/heartbeat", `{"instance_id":""}`, 400},
		{"POST", "/api/mesh/heartbeat", `{"instance_id":"invalid"}`, 404},
		{"GET", "/api/mesh/services", "", 200},
		{"POST", "/api/mesh/deregister", `{"instance_id":"i1"}`, 200},
		{"POST", "/api/mesh/deregister", `invalid_json`, 400},
		{"POST", "/api/mesh/deregister", `{"instance_id":""}`, 400},
		{"POST", "/api/mesh/deregister", `{"instance_id":"invalid"}`, 404},
		{"GET", "/api/health", "", 200},
	}

	for _, tt := range tests {
		req := httptest.NewRequest(tt.method, tt.path, bytes.NewBufferString(tt.body))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		if w.Code != tt.code {
			t.Errorf("%s %s expected %d, got %d", tt.method, tt.path, tt.code, w.Code)
		}
	}
}
