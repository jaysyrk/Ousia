package controlplane

import (
	"testing"

	"github.com/jaysyrk/ousia/internal/balancer"
	"github.com/jaysyrk/ousia/internal/router"
	"github.com/jaysyrk/ousia/pkg/config"
)

func TestBuildUpdateFunc(t *testing.T) {
	r := router.New(nil)
	balancers := make(map[string]balancer.Balancer)
	fn := BuildUpdateFunc(r, balancers)

	cfg := &config.OusiaConfig{
		VirtualHosts: []config.VirtualHostConfig{
			{
				Hostname: "example.com",
				TLS:      &config.TLSConfig{CertFile: "c", KeyFile: "k"},
				Routes: []config.RouteConfig{
					{
						ID:       "r1",
						Priority: 1,
						Match:    config.MatchConfig{PathPrefix: "/"},
						Action:   config.ActionConfig{Upstream: "u1"},
					},
				},
			},
		},
		Upstreams: []config.UpstreamConfig{
			{
				Name:      "u1",
				Algorithm: "round-robin",
				Endpoints: []config.EndpointConfig{
					{ID: "e1", Address: "127.0.0.1:80", Weight: 0},
				},
			},
		},
	}

	fn(cfg)

	if len(balancers) != 1 {
		t.Error("expected 1 balancer")
	}
}
