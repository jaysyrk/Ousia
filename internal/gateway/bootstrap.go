package gateway

import (
	"fmt"

	"github.com/jaysyrk/ousia/internal/balancer"
	"github.com/jaysyrk/ousia/internal/router"
	"github.com/jaysyrk/ousia/pkg/config"
	"github.com/jaysyrk/ousia/pkg/types"
)

func Bootstrap(cfg *config.OusiaConfig) (*Server, error) {
	virtualHosts, err := buildVirtualHosts(cfg)
	if err != nil {
		return nil, err
	}

	balancers, err := buildBalancers(cfg)
	if err != nil {
		return nil, err
	}

	r := router.New(virtualHosts)
	h := NewHandler(r, balancers)
	s := NewServer(cfg.Gateway.ListenAddr, h)

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

func buildBalancers(cfg *config.OusiaConfig) (map[string]balancer.Balancer, error) {
	balancers := make(map[string]balancer.Balancer)

	for _, upCfg := range cfg.Upstreams {
		var endpoints []*types.Endpoint

		for _, epCfg := range upCfg.Endpoints {
			w := epCfg.Weight
			if w == 0 {
				w = 1
			}
			endpoints = append(endpoints, &types.Endpoint{
				ID:       epCfg.ID,
				Address:  epCfg.Address,
				Weight:   w,
				Healthy:  true,
				Metadata: epCfg.Meta,
			})
		}

		pool := &types.UpstreamPool{
			Name:      upCfg.Name,
			Endpoints: endpoints,
			Algorithm: types.LBAlgorithm(upCfg.Algorithm),
		}

		lb, err := balancer.New(pool)
		if err != nil {
			return nil, fmt.Errorf("bootstrap: %w", err)
		}

		balancers[upCfg.Name] = lb
	}

	return balancers, nil
}
