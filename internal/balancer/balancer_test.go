package balancer

import (
	"testing"

	"github.com/jaysyrk/ousia/pkg/types"
)

func healthyEndpoints() []*types.Endpoint {
	return []*types.Endpoint{
		{ID: "ep-1", Address: "10.0.0.1:8080", Weight: 1, Healthy: true},
		{ID: "ep-2", Address: "10.0.0.2:8080", Weight: 1, Healthy: true},
		{ID: "ep-3", Address: "10.0.0.3:8080", Weight: 1, Healthy: true},
	}
}

func TestRoundRobin_Distributes(t *testing.T) {
	rr := NewRoundRobin(healthyEndpoints())
	counts := map[string]int{}

	for i := 0; i < 9; i++ {
		ep, err := rr.Next("")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		counts[ep.ID]++
	}

	for _, id := range []string{"ep-1", "ep-2", "ep-3"} {
		if counts[id] != 3 {
			t.Errorf("expected ep %s to get 3 requests, got %d", id, counts[id])
		}
	}
}

func TestRoundRobin_SkipsUnhealthy(t *testing.T) {
	endpoints := healthyEndpoints()
	endpoints[1].Healthy = false
	rr := NewRoundRobin(endpoints)

	for i := 0; i < 6; i++ {
		ep, err := rr.Next("")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ep.ID == "ep-2" {
			t.Fatal("got unhealthy endpoint ep-2")
		}
	}
}

func TestRoundRobin_NoHealthy(t *testing.T) {
	endpoints := healthyEndpoints()
	for _, ep := range endpoints {
		ep.Healthy = false
	}
	rr := NewRoundRobin(endpoints)
	_, err := rr.Next("")
	if err != ErrNoHealthyEndpoints {
		t.Fatalf("expected ErrNoHealthyEndpoints, got %v", err)
	}
}

func TestWRR_RespectsWeights(t *testing.T) {
	endpoints := []*types.Endpoint{
		{ID: "ep-1", Address: "10.0.0.1:8080", Weight: 3, Healthy: true},
		{ID: "ep-2", Address: "10.0.0.2:8080", Weight: 1, Healthy: true},
	}
	wrr := NewWRR(endpoints)
	counts := map[string]int{}

	for i := 0; i < 40; i++ {
		ep, err := wrr.Next("")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		counts[ep.ID]++
	}

	if counts["ep-1"] != 30 {
		t.Errorf("expected ep-1 to get 30 requests, got %d", counts["ep-1"])
	}
	if counts["ep-2"] != 10 {
		t.Errorf("expected ep-2 to get 10 requests, got %d", counts["ep-2"])
	}
}

func TestLeastConn_PicksLowest(t *testing.T) {
	endpoints := healthyEndpoints()
	lc := NewLeastConn(endpoints)

	lc.entries[0].active.Store(5)
	lc.entries[1].active.Store(3)
	lc.entries[2].active.Store(1)

	ep, err := lc.Next("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ep.ID != "ep-3" {
		t.Errorf("expected ep-3 (least connections), got %s", ep.ID)
	}
}
