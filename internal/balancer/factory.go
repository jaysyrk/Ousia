package balancer

import (
	"fmt"

	"github.com/jaysyrk/ousia/pkg/types"
)

func New(pool *types.UpstreamPool) (Balancer, error) {
	switch pool.Algorithm {
	case types.AlgoRoundRobin, "":
		return NewRoundRobin(pool.Endpoints), nil
	case types.AlgoWRR:
		return NewWRR(pool.Endpoints), nil
	case types.AlgoLeastConn:
		return NewLeastConn(pool.Endpoints), nil
	default:
		return nil, fmt.Errorf("balancer: unknown algorithm %q", pool.Algorithm)
	}
}
