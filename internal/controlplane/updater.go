package controlplane

import (
	"fmt"
	"time"

	"github.com/jaysyrk/ousia/internal/balancer"
	"github.com/jaysyrk/ousia/internal/router"
	"github.com/jaysyrk/ousia/pkg/config"
	"github.com/jaysyrk/ousia/pkg/types"
)

func BuildUpdateFunc(r *router.Router, balancers map[string]balancer.Balancer) UpdateFunc {
	return func(cfg *config.OusiaConfig) {
		applyVirtualHosts(r, cfg)
		applyUpstreams(balancers, cfg)
	}
}

func applyVirtualHosts(r *router.Router, cfg *config.OusiaConfig) {
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
			timeout, _ := time.ParseDuration(rCfg.Action.Timeout)
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
					Timeout:           timeout,
					RetryCount:        rCfg.Action.RetryCount,
				},
			}
			vh.Routes = append(vh.Routes, route)
		}

		r.AddVirtualHost(vh)
		fmt.Printf("updater: applied virtual host %q\n", vh.Hostname)
	}
}

func applyUpstreams(balancers map[string]balancer.Balancer, cfg *config.OusiaConfig) {
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

		pool := &types.UpstreamPool{
			Name:		upCfg.Name,
			Endpoints:	endpoints,
			Algorithm:	types.LBAlgorithm(upCfg.Algorithm),
		}

		lb, err := balancer.New(pool)
		if err != nil {
			fmt.Printf("updater: skipping upstream %q: %v\n", upCfg.Name, err)
			continue
		}

		balancers[upCfg.Name] = lb
		fmt.Printf("updater: applied upstream %q with %d endpoint(s)\n", upCfg.Name, len(endpoints))
	}
}
