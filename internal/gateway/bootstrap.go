package gateway

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"github.com/jaysyrk/ousia/internal/balancer"
	"github.com/jaysyrk/ousia/internal/controlplane"
	"github.com/jaysyrk/ousia/internal/observability"
	"github.com/jaysyrk/ousia/internal/router"
	"github.com/jaysyrk/ousia/pkg/config"
	"github.com/jaysyrk/ousia/pkg/healthcheck"
	"github.com/jaysyrk/ousia/pkg/types"
)

func Bootstrap(cfg *config.OusiaConfig, configPath string) (*Server, error) {
	observability.InitLogger()
	observability.InitMetrics()

	virtualHosts, err := buildVirtualHosts(cfg)
	if err != nil {
		return nil, err
	}

	balancers, allEndpointsByPool, err := buildBalancers(cfg)
	if err != nil {
		return nil, err
	}

	for _, upCfg := range cfg.Upstreams {
		endpoints := allEndpointsByPool[upCfg.Name]
		hcCfg := buildHealthCheckConfig(upCfg.HealthCheck)
		hc := healthcheck.New(endpoints, hcCfg)
		hc.Start(context.Background())
	}

	r := router.New(virtualHosts)
	h := NewHandler(r, balancers)

	var tlsCfg *tls.Config
	if cfg.Gateway.TLSAddr != "" {
		for _, vh := range cfg.VirtualHosts {
			if vh.TLS != nil {
				tlsCfg, err = buildTLSConfig(&types.TLSConfig{
					CertFile: vh.TLS.CertFile,
					KeyFile:  vh.TLS.KeyFile,
				})
				if err != nil {
					return nil, err
				}
				break
			}
		}
	}

	s := NewServer(cfg, r, balancers, h, tlsCfg)

	store := controlplane.NewStore(cfg)
	watcher := controlplane.NewWatcher(configPath, store, 5*time.Second)
	watcher.OnChange(controlplane.BuildUpdateFunc(r, balancers))

	mesh := controlplane.NewMeshRegistry(30 * time.Second)
	mesh.StartCleanup(context.Background())
	admin := controlplane.NewAdminAPI(r, balancers, store, mesh)
	observability.StartAdminServer(cfg.Gateway.AdminAddr, admin.RegisterRoutes)

	go watcher.Start(context.Background())

	return s, nil
}

func buildHealthCheckConfig(hcCfg config.HealthCheckConfig) healthcheck.Config {
	cfg := healthcheck.DefaultConfig()
	if hcCfg.Path != "" {
		cfg.Path = hcCfg.Path
	}
	if hcCfg.Interval != "" {
		if d, err := time.ParseDuration(hcCfg.Interval); err == nil {
			cfg.Interval = d
		}
	}
	if hcCfg.Timeout != "" {
		if d, err := time.ParseDuration(hcCfg.Timeout); err == nil {
			cfg.Timeout = d
		}
	}
	if hcCfg.FailThreshold > 0 {
		cfg.FailThreshold = hcCfg.FailThreshold
	}
	if hcCfg.SuccessThreshold > 0 {
		cfg.SuccessThreshold = hcCfg.SuccessThreshold
	}
	return cfg
}

func buildVirtualHosts(cfg *config.OusiaConfig) ([]*types.VirtualHost, error) {
	var hosts []*types.VirtualHost

	for _, vhCfg := range cfg.VirtualHosts {
		vh := &types.VirtualHost{
			Hostname: vhCfg.Hostname,
		}

		if vhCfg.TLS != nil {
			vh.TLS = &types.TLSConfig{
				CertFile:	vhCfg.TLS.CertFile,
				KeyFile:	vhCfg.TLS.KeyFile,
			}
		}

		for _, rCfg := range vhCfg.Routes {
			route := &types.Route{
				ID:       rCfg.ID,
				Priority: rCfg.Priority,
				Match: types.RouteMatch{
					PathPrefix: rCfg.Match.PathPrefix,
					PathExact:  rCfg.Match.PathExact,
					Methods:    rCfg.Match.Methods,
					Headers:    rCfg.Match.Headers,
				},
				Action: types.RouteAction{
					UpstreamPool:      rCfg.Action.Upstream,
					StripPrefix:       rCfg.Action.StripPrefix,
					AddHeaders:        rCfg.Action.AddHeaders,
					RemoveHeaders:     rCfg.Action.RemoveHeaders,
					AddRespHeaders:    rCfg.Action.AddRespHeaders,
					RemoveRespHeaders: rCfg.Action.RemoveRespHeaders,
					Timeout:           parseDurationSafe(rCfg.Action.Timeout),
					RetryCount:        rCfg.Action.RetryCount,
				},
			}
			vh.Routes = append(vh.Routes, route)
		}

		hosts = append(hosts, vh)
	}

	return hosts, nil
}

func buildBalancers(cfg *config.OusiaConfig) (map[string]balancer.Balancer, map[string][]*types.Endpoint, error) {
	balancers := make(map[string]balancer.Balancer)
	endpointsByPool := make(map[string][]*types.Endpoint)

	for _, upCfg := range cfg.Upstreams {
		var endpoints []*types.Endpoint

		for _, epCfg := range upCfg.Endpoints {
			w := epCfg.Weight
			if w == 0 {
				w = 1
			}
			ep := &types.Endpoint{
				ID:       epCfg.ID,
				Address:  epCfg.Address,
				Weight:   w,
				Metadata: epCfg.Meta,
			}
			ep.Healthy.Store(true)
			endpoints = append(endpoints, ep)
		}

		endpointsByPool[upCfg.Name] = endpoints

		pool := &types.UpstreamPool{
			Name:      upCfg.Name,
			Endpoints: endpoints,
			Algorithm: types.LBAlgorithm(upCfg.Algorithm),
		}

		lb, err := balancer.New(pool)
		if err != nil {
			return nil, nil, fmt.Errorf("bootstrap: %w", err)
		}

		balancers[upCfg.Name] = lb
	}

	return balancers, endpointsByPool, nil
}

func parseDurationSafe(s string) time.Duration {
	if s == "" {
		return 0
	}
	d, _ := time.ParseDuration(s)
	return d
}
