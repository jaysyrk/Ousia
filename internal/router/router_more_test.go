package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jaysyrk/ousia/pkg/types"
)

func TestRouter_HeaderMatch(t *testing.T) {
	vh := &types.VirtualHost{
		Hostname: "*",
		Routes: []*types.Route{
			{
				ID: "with-header",
				Match: types.RouteMatch{
					PathExact: "/test",
					Headers: map[string]string{
						"X-Custom": "true",
					},
				},
			},
		},
	}

	r := New([]*types.VirtualHost{vh})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	// Missing header
	_, _, err := r.Match(req)
	if err == nil {
		t.Fatal("expected no match, but got one")
	}

	// With header
	req.Header.Set("X-Custom", "true")
	route, _, err := r.Match(req)
	if err != nil {
		t.Fatalf("expected match, got error: %v", err)
	}
	if route.ID != "with-header" {
		t.Fatalf("expected 'with-header', got %q", route.ID)
	}
}

func TestRouter_AddRemoveVirtualHost(t *testing.T) {
	r := New(nil)

	vh := &types.VirtualHost{
		Hostname: "api.example.com",
		Routes: []*types.Route{
			{ID: "r2", Priority: 2},
			{ID: "r1", Priority: 1},
		},
	}

	r.AddVirtualHost(vh)
	
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Host = "api.example.com"
	_, _, err := r.Match(req)
	if err != nil && err.Error() == "router: no virtual host for \"api.example.com\"" {
		t.Fatalf("expected to find virtual host")
	}

	r.RemoveVirtualHost("api.example.com")
	_, _, err = r.Match(req)
	if err == nil || err.Error() != "router: no virtual host for \"api.example.com\"" {
		t.Fatalf("expected no virtual host error")
	}
}

func TestRouter_NormalizeHost(t *testing.T) {
	vh := &types.VirtualHost{
		Hostname: "example.com",
		Routes: []*types.Route{
			{ID: "root", Match: types.RouteMatch{PathExact: "/"}},
		},
	}

	r := New([]*types.VirtualHost{vh})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Host = "example.com:8080"

	route, _, err := r.Match(req)
	if err != nil {
		t.Fatalf("expected match with port stripped, got %v", err)
	}
	if route.ID != "root" {
		t.Fatalf("expected 'root', got %q", route.ID)
	}
}
