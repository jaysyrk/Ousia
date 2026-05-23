package balancer

import (
	"sync"
	"sync/atomic"

	"github.com/jaysyrk/ousia/pkg/types"
)

type connEndpoint struct {
	ep      *types.Endpoint
	active  atomic.Int64
}

type LeastConn struct {
	mu      sync.RWMutex
	entries []*connEndpoint
}

func NewLeastConn(endpoints []*types.Endpoint) *LeastConn {
	lc := &LeastConn{}
	for _, ep := range endpoints {
		lc.entries = append(lc.entries, &connEndpoint{ep: ep})
	}
	return lc
}

func (lc *LeastConn) Next(key string) (*types.Endpoint, error) {
	lc.mu.RLock()
	defer lc.mu.RUnlock()

	var best *connEndpoint
	for _, ce := range lc.entries {
		if !ce.ep.Healthy {
			continue
		}
		if best == nil || ce.active.Load() < best.active.Load() {
			best = ce
		}
	}

	if best == nil {
		return nil, ErrNoHealthyEndpoints
	}

	best.active.Add(1)
	return best.ep, nil
}

func (lc *LeastConn) Done(id string) {
	lc.mu.RLock()
	defer lc.mu.RUnlock()
	for _, ce := range lc.entries {
		if ce.ep.ID == id {
			ce.active.Add(-1)
			return
		}
	}
}

func (lc *LeastConn) Add(ep *types.Endpoint) {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	lc.entries = append(lc.entries, &connEndpoint{ep: ep})
}

func (lc *LeastConn) Remove(id string) {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	filtered := lc.entries[:0]
	for _, ce := range lc.entries {
		if ce.ep.ID != id {
			filtered = append(filtered, ce)
		}
	}
	lc.entries = filtered
}