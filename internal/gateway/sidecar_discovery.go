package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type ServiceMapper struct {
	mu        sync.RWMutex
	endpoints map[string][]string
	counters  map[string]int
}

func NewServiceMapper() *ServiceMapper {
	return &ServiceMapper{
		endpoints: make(map[string][]string),
		counters:  make(map[string]int),
	}
}

func (m *ServiceMapper) Update(service string, targets []string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.endpoints[service] = append([]string(nil), targets...)
	m.counters[service] = 0
}

func (m *ServiceMapper) Next(service string) (string, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	targets, ok := m.endpoints[service]
	if !ok || len(targets) == 0 {
		return "", false
	}
	index := m.counters[service] % len(targets)
	m.counters[service] = index + 1
	return targets[index], true
}

type SidecarDiscovery struct {
	adminURL string
	mapper   *ServiceMapper
	client   *http.Client
	interval time.Duration
}

func NewSidecarDiscovery(adminURL string, mapper *ServiceMapper, interval time.Duration) *SidecarDiscovery {
	return &SidecarDiscovery{
		adminURL: adminURL,
		mapper:   mapper,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
		interval: interval,
	}
}

func (d *SidecarDiscovery) Start(ctx context.Context) {
	d.refresh(ctx)
	ticker := time.NewTicker(d.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			d.refresh(ctx)
		}
	}
}

func (d *SidecarDiscovery) refresh(ctx context.Context) {
	upstreams, err := d.fetchUpstreams(ctx)
	if err != nil {
		fmt.Printf("sidecar discovery refresh failed: %v\n", err)
		return
	}

	for _, upstream := range upstreams {
		var targets []string
		for _, endpoint := range upstream.Endpoints {
			if endpoint.Healthy {
				targets = append(targets, endpoint.Address)
			}
		}
		d.mapper.Update(upstream.Name, targets)
	}
}

func (d *SidecarDiscovery) fetchUpstreams(ctx context.Context) ([]upstreamResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, d.adminURL+"/api/upstreams", nil)
	if err != nil {
		return nil, err
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("admin returned %d", resp.StatusCode)
	}

	var payload struct {
		Upstreams []upstreamResponse `json:"upstreams"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	return payload.Upstreams, nil
}

type upstreamResponse struct {
	Name      string             `json:"name"`
	Endpoints []endpointResponse `json:"endpoints"`
}

type endpointResponse struct {
	ID      string `json:"id"`
	Address string `json:"address"`
	Weight  int    `json:"weight"`
	Healthy bool   `json:"healthy"`
}
