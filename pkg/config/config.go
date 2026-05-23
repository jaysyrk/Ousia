package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type OusiaConfig struct {
	Gateway  GatewayConfig   `yaml:"gateway"`
	VirtualHosts []VirtualHostConfig `yaml:"virtual_hosts"`
	Upstreams []UpstreamConfig `yaml:"upstreams"`
}

type GatewayConfig struct {
	ListenAddr string `yaml:"listen_addr"`
	TLSAddr string `yaml:"tls_addr"`
	AdminAddr string `yaml:"admin_addr"`
}

type VirtualHostConfig struct {
	Hostname string        `yaml:"hostname"`
	TLS      *TLSConfig    `yaml:"tls,omitempty"`
	Routes   []RouteConfig `yaml:"routes"`
}

type TLSConfig struct {
	CertFile string `yaml:"cert_file"`
	KeyFile  string `yaml:"key_file"`
}

type RouteConfig struct {
	ID       string       `yaml:"id"`
	Priority int          `yaml:"priority"`
	Match    MatchConfig  `yaml:"match"`
	Action   ActionConfig `yaml:"action"`
}

type MatchConfig struct {
	PathPrefix string            `yaml:"path_prefix,omitempty"`
	PathExact  string            `yaml:"path_exact,omitempty"`
	Methods    []string          `yaml:"methods,omitempty"`
	Headers    map[string]string `yaml:"headers,omitempty"`
}

type ActionConfig struct {
	Upstream    string            `yaml:"upstream"`
	StripPrefix string            `yaml:"strip_prefix,omitempty"`
	AddHeaders  map[string]string `yaml:"add_headers,omitempty"`
	Timeout     string            `yaml:"timeout,omitempty"`
	RetryCount  int               `yaml:"retry_count,omitempty"`
}

type UpstreamConfig struct {
	Name      string           `yaml:"name"`
	Algorithm string           `yaml:"algorithm"`
	Endpoints []EndpointConfig `yaml:"endpoints"`
}

type EndpointConfig struct {
	ID      string            `yaml:"id"`
	Address string            `yaml:"address"`
	Weight  int               `yaml:"weight,omitempty"`
	Meta    map[string]string `yaml:"meta,omitempty"`
}

func Load(path string) (*OusiaConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("config: cannot read file %q: %w", path, err)
	}

	var cfg OusiaConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("config: invalid yaml: %w", err)
	}

	if err := validate(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func validate(cfg *OusiaConfig) error {
	if cfg.Gateway.ListenAddr == "" {
		return fmt.Errorf("config: gateway.listen_addr is required")
	}
	for _, vh := range cfg.VirtualHosts {
		if vh.Hostname == "" {
			return fmt.Errorf("config: a virtual_host is missing its hostname")
		}
		for _, r := range vh.Routes {
			if r.ID == "" {
				return fmt.Errorf("config: a route under %q is missing its id", vh.Hostname)
			}
			if r.Action.Upstream == "" {
				return fmt.Errorf("config: route %q has no upstream", r.ID)
			}
			if r.Action.Timeout != "" {
				if _, err := time.ParseDuration(r.Action.Timeout); err != nil {
					return fmt.Errorf("config: route %q has invalid timeout %q", r.ID, r.Action.Timeout)
				}
			}
		}
	}
	for _, u := range cfg.Upstreams {
		if u.Name == "" {
			return fmt.Errorf("config: an upstream is missing its name")
		}
		if len(u.Endpoints) == 0 {
			return fmt.Errorf("config: upstream %q has no endpoints", u.Name)
		}
	}
	return nil
}