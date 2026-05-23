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
	observability.StartAdminServer(cfg.Gateway.AdminAddr)

	virtualHosts, err := buildVirtualHosts(cfg)
	if err != nil {
		return nil, err
	}

	balancers, allEndpoints, err := buildBalancers(cfg)
	if err != nil {
		return nil, err
	}

	hc := healthcheck.New(allEndpoints, healthcheck.DefaultConfig())
	hc.Start(context.Background())

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

	s := NewServer(cfg.Gateway.ListenAddr, cfg.Gateway.TLSAddr, h, tlsCfg)

	store := controlplane.NewStore(cfg)
	watcher := controlplane.NewWatcher(configPath, store, 5*time.Second)
	watcher.OnChange(controlplane.BuildUpdateFunc(r, balancers))

	go watcher.Start(context.Background())

	return s, nil
}

func buildVirtualHosts(cfg *config.OusiaConfig) ([]*types.VirtualHost, error) {
	var hosts []*types.VirtualHost

	for _, vhCfg := range cfg.VirtualHosts {
		vh := &types.VirtualHost{
			Hostname: vhCfg.Hostname,
		}

		if vhCfg.TLS != nil {
			vh.TLS = &types.TLSConfig{
				CertFile: vhCfg.TLS.CertFile,
				KeyFile:  vhCfg.TLS.KeyFile,
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
					UpstreamPool: rCfg.Action.Upstream,
					StripPrefix:  rCfg.Action.StripPrefix,
					AddHeaders:   rCfg.Action.AddHeaders,
					RetryCount:   rCfg.Action.RetryCount,
				},
			}
			vh.Routes = append(vh.Routes, route)
		}

		hosts = append(hosts, vh)
	}

	return hosts, nil
}

func buildBalancers(cfg *config.OusiaConfig) (map[string]balancer.Balancer, []*types.Endpoint, error) {
	balancers := make(map[string]balancer.Balancer)
	var allEndpoints []*types.Endpoint

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
				Healthy:  true,
				Metadata: epCfg.Meta,
			}
			endpoints = append(endpoints, ep)
			allEndpoints = append(allEndpoints, ep)
		}

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

	return balancers, allEndpoints, nil
}
