package balancer

import (
	"errors"

	"github.com/jaysyrk/ousia/pkg/types"
)

var ErrNoHealthyEndpoints = errors.New("balancer: no healthy endpoints available")

type Balancer interface {
	Next(key string) (*types.Endpoint, error)
	Add(ep *types.Endpoint)
	Remove(id string)
}

func healthy(endpoints []*types.Endpoint) []*types.Endpoint {
	out := make([]*types.Endpoint, 0, len(endpoints))
	for _, ep := range endpoints {
		if ep.Healthy {
			out = append(out, ep)
		}
	}
	return out
}