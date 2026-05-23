package healthcheck

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/jaysyrk/ousia/pkg/types"
)

type Config struct {
	Path          string
	Interval      time.Duration
	Timeout       time.Duration
	FailThreshold int
}

func DefaultConfig() Config {
	return Config{
		Path:          "/healthz",
		Interval:      10 * time.Second,
		Timeout:       2 * time.Second,
		FailThreshold: 2,
	}
}

type Checker struct {
	cfg       Config
	endpoints []*types.Endpoint
	mu        sync.Mutex
	client    *http.Client
}

func New(endpoints []*types.Endpoint, cfg Config) *Checker {
	return &Checker{
		cfg:       cfg,
		endpoints: endpoints,
		client: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

func (c *Checker) Start(ctx context.Context) {
	for _, ep := range c.endpoints {
		go c.watch(ctx, ep)
	}
}

func (c *Checker) Add(ep *types.Endpoint) {
	c.mu.Lock()
	c.endpoints = append(c.endpoints, ep)
	c.mu.Unlock()
}

func (c *Checker) watch(ctx context.Context, ep *types.Endpoint) {
	ticker := time.NewTicker(c.cfg.Interval)
	defer ticker.Stop()

	failures := 0

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			err := c.probe(ep)
			if err != nil {
				failures++
				if failures >= c.cfg.FailThreshold {
					if ep.Healthy {
						ep.Healthy = false
						fmt.Printf("healthcheck: endpoint %s (%s) marked unhealthy: %v\n", ep.ID, ep.Address, err)
					}
				}
			} else {
				if !ep.Healthy {
					fmt.Printf("healthcheck: endpoint %s (%s) recovered\n", ep.ID, ep.Address)
				}
				failures = 0
				ep.Healthy = true
			}
		}
	}
}

func (c *Checker) probe(ep *types.Endpoint) error {
	url := fmt.Sprintf("http://%s%s", ep.Address, c.cfg.Path)
	resp, err := c.client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		return fmt.Errorf("unhealthy status %d", resp.StatusCode)
	}

	return nil
}
