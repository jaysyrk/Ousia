package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jaysyrk/ousia/pkg/types"
)

func TestRouter_ExactPath(t *testing.T) {
	vh := &types.VirtualHost{
		Hostname:	"api.example.com",
		Routes: []*types.Route{
			{
				ID:		"health",
				Priority:	0,
				Match:		types.RouteMatch{PathExact: "/healthz"},
				Action:		types.RouteAction{UpstreamPool: "backend"},
			},
		},
	}

	r := New([]*types.VirtualHost{vh})

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.Host = "api.example.com"

	route, _, err := r.Match(req)
	if err != nil {
		t.Fatalf("expected match, got error: %v", err)
	}
	if route.ID != "health" {
		t.Fatalf("expected route 'health', got %q", route.ID)
	}
}

func TestRouter_PrefixPath(t *testing.T) {
	vh := &types.VirtualHost{
		Hostname:	"api.example.com",
		Routes: []*types.Route{
			{
				ID:		"api-v1",
				Priority:	1,
				Match:		types.RouteMatch{PathPrefix: "/api/v1"},
				Action:		types.RouteAction{UpstreamPool: "backend"},
			},
		},
	}

	r := New([]*types.VirtualHost{vh})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	req.Host = "api.example.com"

	route, _, err := r.Match(req)
	if err != nil {
		t.Fatalf("expected match, got error: %v", err)
	}
	if route.ID != "api-v1" {
		t.Fatalf("expected route 'api-v1', got %q", route.ID)
	}
}

func TestRouter_MethodFilter(t *testing.T) {
	vh := &types.VirtualHost{
		Hostname:	"*",
		Routes: []*types.Route{
			{
				ID:		"post-only",
				Priority:	0,
				Match:		types.RouteMatch{PathPrefix: "/submit", Methods: []string{"POST"}},
				Action:		types.RouteAction{UpstreamPool: "backend"},
			},
		},
	}

	r := New([]*types.VirtualHost{vh})

	req := httptest.NewRequest(http.MethodGet, "/submit", nil)
	req.Host = "anything.com"
	_, _, err := r.Match(req)
	if err == nil {
		t.Fatal("expected no match for GET, but got a match")
	}

	req = httptest.NewRequest(http.MethodPost, "/submit", nil)
	req.Host = "anything.com"
	route, _, err := r.Match(req)
	if err != nil {
		t.Fatalf("expected match for POST, got error: %v", err)
	}
	if route.ID != "post-only" {
		t.Fatalf("expected route 'post-only', got %q", route.ID)
	}
}

func TestRouter_NoMatch(t *testing.T) {
	vh := &types.VirtualHost{
		Hostname:	"api.example.com",
		Routes: []*types.Route{
			{
				ID:	"only-route",
				Match:	types.RouteMatch{PathExact: "/only"},
			},
		},
	}

	r := New([]*types.VirtualHost{vh})

	req := httptest.NewRequest(http.MethodGet, "/other", nil)
	req.Host = "api.example.com"

	_, _, err := r.Match(req)
	if err == nil {
		t.Fatal("expected no match, but got one")
	}
}
