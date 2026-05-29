package balancer

import (
	"sync"
	"sync/atomic"

	"github.com/jaysyrk/ousia/pkg/types"
)

type RoundRobin struct {
	mu		sync.RWMutex
	endpoints	[]*types.Endpoint
	counter		atomic.Uint64
}

func NewRoundRobin(endpoints []*types.Endpoint) *RoundRobin {
	return &RoundRobin{endpoints: endpoints}
}

func (rr *RoundRobin) Next(key string) (*types.Endpoint, error) {
	rr.mu.RLock()
	pool := healthy(rr.endpoints)
	rr.mu.RUnlock()

	if len(pool) == 0 {
		return nil, ErrNoHealthyEndpoints
	}

	idx := rr.counter.Add(1) - 1
	return pool[idx%uint64(len(pool))], nil
}

func (rr *RoundRobin) Add(ep *types.Endpoint) {
	rr.mu.Lock()
	defer rr.mu.Unlock()
	rr.endpoints = append(rr.endpoints, ep)
}

func (rr *RoundRobin) Remove(id string) {
	rr.mu.Lock()
	defer rr.mu.Unlock()
	filtered := rr.endpoints[:0]
	for _, ep := range rr.endpoints {
		if ep.ID != id {
			filtered = append(filtered, ep)
		}
	}
	rr.endpoints = filtered
}

func (rr *RoundRobin) Endpoints() []*types.Endpoint {
	rr.mu.RLock()
	defer rr.mu.RUnlock()
	return rr.endpoints
}

func (rr *RoundRobin) Done(id string) {}

