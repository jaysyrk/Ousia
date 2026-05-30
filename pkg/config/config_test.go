package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	dir := t.TempDir()
	validConfig := filepath.Join(dir, "valid.yaml")
	invalidYAML := filepath.Join(dir, "invalid.yaml")

	os.WriteFile(validConfig, []byte(`
gateway:
  listen_addr: ":8080"
virtual_hosts:
  - hostname: "example.com"
    routes:
      - id: "route1"
        action:
          upstream: "upstream1"
upstreams:
  - name: "upstream1"
    endpoints:
      - id: "ep1"
        address: "127.0.0.1:9090"
`), 0644)

	os.WriteFile(invalidYAML, []byte(`
gateway:
  listen_addr: ":8080"
  invalid
    yaml
`), 0644)

	t.Run("Valid file", func(t *testing.T) {
		cfg, err := Load(validConfig)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if cfg.Gateway.ListenAddr != ":8080" {
			t.Errorf("expected :8080, got %s", cfg.Gateway.ListenAddr)
		}
	})

	t.Run("Invalid yaml file", func(t *testing.T) {
		_, err := Load(invalidYAML)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("Missing file", func(t *testing.T) {
		_, err := Load(filepath.Join(dir, "missing.yaml"))
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *OusiaConfig
		wantErr bool
	}{
		{
			name: "Valid config",
			cfg: &OusiaConfig{
				Gateway: GatewayConfig{ListenAddr: ":8080"},
				VirtualHosts: []VirtualHostConfig{
					{
						Hostname: "example.com",
						Routes: []RouteConfig{
							{ID: "r1", Action: ActionConfig{Upstream: "u1"}},
						},
					},
				},
				Upstreams: []UpstreamConfig{
					{Name: "u1", Endpoints: []EndpointConfig{{ID: "e1", Address: "127.0.0.1:80"}}},
				},
			},
			wantErr: false,
		},
		{
			name: "Missing gateway listen addr",
			cfg: &OusiaConfig{
				Gateway: GatewayConfig{ListenAddr: ""},
			},
			wantErr: true,
		},
		{
			name: "Missing virtual host hostname",
			cfg: &OusiaConfig{
				Gateway: GatewayConfig{ListenAddr: ":8080"},
				VirtualHosts: []VirtualHostConfig{
					{Hostname: ""},
				},
			},
			wantErr: true,
		},
		{
			name: "Missing route id",
			cfg: &OusiaConfig{
				Gateway: GatewayConfig{ListenAddr: ":8080"},
				VirtualHosts: []VirtualHostConfig{
					{
						Hostname: "example.com",
						Routes: []RouteConfig{
							{ID: "", Action: ActionConfig{Upstream: "u1"}},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Missing route upstream",
			cfg: &OusiaConfig{
				Gateway: GatewayConfig{ListenAddr: ":8080"},
				VirtualHosts: []VirtualHostConfig{
					{
						Hostname: "example.com",
						Routes: []RouteConfig{
							{ID: "r1", Action: ActionConfig{Upstream: ""}},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Invalid route timeout",
			cfg: &OusiaConfig{
				Gateway: GatewayConfig{ListenAddr: ":8080"},
				VirtualHosts: []VirtualHostConfig{
					{
						Hostname: "example.com",
						Routes: []RouteConfig{
							{ID: "r1", Action: ActionConfig{Upstream: "u1", Timeout: "invalid"}},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Missing upstream name",
			cfg: &OusiaConfig{
				Gateway: GatewayConfig{ListenAddr: ":8080"},
				Upstreams: []UpstreamConfig{
					{Name: "", Endpoints: []EndpointConfig{{ID: "e1", Address: "127.0.0.1:80"}}},
				},
			},
			wantErr: true,
		},
		{
			name: "Missing upstream endpoints",
			cfg: &OusiaConfig{
				Gateway: GatewayConfig{ListenAddr: ":8080"},
				Upstreams: []UpstreamConfig{
					{Name: "u1", Endpoints: []EndpointConfig{}},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
