package gateway

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jaysyrk/ousia/pkg/config"
)

func TestBootstrap(t *testing.T) {
	cfgFile := t.TempDir() + "/config.yaml"
	os.WriteFile(cfgFile, []byte(`
gateway:
  listen_addr: "127.0.0.1:0"
  admin_addr: "127.0.0.1:0"
virtual_hosts:
  - hostname: "example.com"
    rate_limit:
      requests_per_second: 10
      burst: 20
      key_by: "header:X-User"
    routes:
      - id: "r1"
        match:
          path_prefix: "/"
        action:
          upstream: "u1"
          strip_prefix: "/api"
          add_headers: {"X-Added": "val"}
upstreams:
  - name: "u1"
    algorithm: "round-robin"
    health_check:
      path: "/health"
      interval: "1s"
      timeout: "1s"
    endpoints:
      - id: "e1"
        address: "127.0.0.1:8080"
        weight: 1
`), 0644)

	cfg := &config.OusiaConfig{
		Gateway: config.GatewayConfig{
			ListenAddr: "127.0.0.1:0",
			AdminAddr:  "127.0.0.1:0",
		},
		VirtualHosts: []config.VirtualHostConfig{
			{
				Hostname: "example.com",
				RateLimit: &config.RateLimitConfig{
					RequestsPerSecond: 10,
					Burst:             20,
					KeyBy:             "header:X-User",
				},
				Routes: []config.RouteConfig{
					{
						ID: "r1",
						Match: config.MatchConfig{PathPrefix: "/"},
						Action: config.ActionConfig{
							Upstream: "u1",
							StripPrefix: "/api",
							AddHeaders: map[string]string{"X-Added": "val"},
						},
					},
				},
			},
		},
		Upstreams: []config.UpstreamConfig{
			{
				Name: "u1",
				Algorithm: "round-robin",
				HealthCheck: config.HealthCheckConfig{
					Path: "/health", Interval: "1s", Timeout: "1s", FailThreshold: 3, SuccessThreshold: 2,
				},
				Endpoints: []config.EndpointConfig{
					{ID: "e1", Address: "127.0.0.1:8080", Weight: 0},
				},
			},
		},
	}

	server, err := Bootstrap(cfg, cfgFile)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if server == nil {
		t.Fatal("expected server to be non-nil")
	}

	go func() {
		_ = server.Start()
	}()

	time.Sleep(50 * time.Millisecond)

	err = server.Shutdown(context.Background())
	if err != nil {
		t.Fatalf("expected graceful shutdown, got %v", err)
	}
}

func TestBootstrap_BadBalancer(t *testing.T) {
	cfg := &config.OusiaConfig{
		Upstreams: []config.UpstreamConfig{
			{Name: "u1", Algorithm: "invalid"},
		},
	}
	_, err := Bootstrap(cfg, "dummy")
	if err == nil {
		t.Fatal("expected error for invalid balancer")
	}
}

func TestTLSConfigError(t *testing.T) {
	cfg := &config.OusiaConfig{
		Gateway: config.GatewayConfig{
			TLSAddr: "127.0.0.1:0",
		},
		VirtualHosts: []config.VirtualHostConfig{
			{
				Hostname: "*",
				TLS: &config.TLSConfig{CertFile: "nonexistent", KeyFile: "nonexistent"},
			},
		},
	}
	_, err := Bootstrap(cfg, "dummy")
	if err == nil {
		t.Fatal("expected error for nonexistent certs")
	}
}
