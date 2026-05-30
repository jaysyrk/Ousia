package balancer

import (
	"testing"

	"github.com/jaysyrk/ousia/pkg/types"
)

func TestNew(t *testing.T) {
	_, err := New(&types.UpstreamPool{Algorithm: types.AlgoRoundRobin})
	if err != nil {
		t.Errorf("expected no error for round-robin")
	}

	_, err = New(&types.UpstreamPool{Algorithm: types.AlgoWRR})
	if err != nil {
		t.Errorf("expected no error for wrr")
	}

	_, err = New(&types.UpstreamPool{Algorithm: types.AlgoLeastConn})
	if err != nil {
		t.Errorf("expected no error for least-conn")
	}

	// default fallback to round-robin
	_, err = New(&types.UpstreamPool{Algorithm: ""})
	if err != nil {
		t.Errorf("expected no error for empty algo")
	}

	_, err = New(&types.UpstreamPool{Algorithm: types.LBAlgorithm("invalid")})
	if err == nil {
		t.Errorf("expected error for invalid algo")
	}
}

func TestRoundRobin_Lifecycle(t *testing.T) {
	rr := NewRoundRobin(nil)
	ep := &types.Endpoint{ID: "e1", Healthy: true}
	rr.Add(ep)
	if len(rr.Endpoints()) != 1 {
		t.Errorf("expected 1 endpoint")
	}
	rr.Done("e1") // no-op
	rr.Remove("e1")
	if len(rr.Endpoints()) != 0 {
		t.Errorf("expected 0 endpoints")
	}
}

func TestWRR_Lifecycle(t *testing.T) {
	wrr := NewWRR(nil)
	ep := &types.Endpoint{ID: "e1", Healthy: true, Weight: 0}
	wrr.Add(ep)
	if len(wrr.Endpoints()) != 1 {
		t.Errorf("expected 1 endpoint")
	}
	_, err := wrr.Next("") // test zero weight to 1
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	wrr.Done("e1") // no-op
	wrr.Remove("e1")
	if len(wrr.Endpoints()) != 0 {
		t.Errorf("expected 0 endpoints")
	}
	_, err = wrr.Next("")
	if err != ErrNoHealthyEndpoints {
		t.Errorf("expected ErrNoHealthyEndpoints")
	}
}

func TestLeastConn_Lifecycle(t *testing.T) {
	lc := NewLeastConn(nil)
	ep := &types.Endpoint{ID: "e1", Healthy: true}
	lc.Add(ep)
	if len(lc.Endpoints()) != 1 {
		t.Errorf("expected 1 endpoint")
	}
	_, err := lc.Next("")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	lc.Done("e1")
	lc.Remove("e1")
	if len(lc.Endpoints()) != 0 {
		t.Errorf("expected 0 endpoints")
	}

	ep2 := &types.Endpoint{ID: "e2", Healthy: false}
	lc.Add(ep2)
	_, err = lc.Next("")
	if err != ErrNoHealthyEndpoints {
		t.Errorf("expected ErrNoHealthyEndpoints")
	}
}
