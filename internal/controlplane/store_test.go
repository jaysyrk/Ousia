package controlplane

import (
	"testing"
	"github.com/jaysyrk/ousia/pkg/config"
)

func TestStore(t *testing.T) {
	initial := &config.OusiaConfig{}
	store := NewStore(initial)
	if store.Get() != initial {
		t.Error("expected initial config")
	}

	updated := &config.OusiaConfig{
		Gateway: config.GatewayConfig{ListenAddr: ":8080"},
	}
	store.Set(updated)
	if store.Get() != updated {
		t.Error("expected updated config")
	}
}
