package balancer

import (
	"sync"
	"sync/atomic"

	"github.com/jaysyrk/ousia/pkg/types"
)

type WRR struct {
	mu      sync.RWMutex
	entries []*types.Endpoint
	counter atomic.Uint64
}

func NewWRR(endpoints []*types.Endpoint) *WRR {
	return &WRR{entries: endpoints}
}

func (w *WRR) Next(key string) (*types.Endpoint, error) {
	w.mu.RLock()
	pool := healthy(w.entries)
	w.mu.RUnlock()

	if len(pool) == 0 {
		return nil, ErrNoHealthyEndpoints
	}

	weighted := make([]*types.Endpoint, 0)
	for _, ep := range pool {
		weight := ep.Weight
		if weight <= 0 {
			weight = 1
		}
		for i := 0; i < weight; i++ {
			weighted = append(weighted, ep)
		}
	}

	idx := w.counter.Add(1) - 1
	return weighted[idx%uint64(len(weighted))], nil
}

func (w *WRR) Add(ep *types.Endpoint) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.entries = append(w.entries, ep)
}

func (w *WRR) Remove(id string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	filtered := w.entries[:0]
	for _, ep := range w.entries {
		if ep.ID != id {
			filtered = append(filtered, ep)
		}
	}
	w.entries = filtered
}

func (w *WRR) Endpoints() []*types.Endpoint {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.entries
}
