package controlplane

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jaysyrk/ousia/pkg/config"
)

func TestWatcher(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	os.WriteFile(path, []byte(`
gateway:
  listen_addr: ":8080"
`), 0644)

	store := NewStore(&config.OusiaConfig{})
	w := NewWatcher(path, store, 10*time.Millisecond)

	var called atomic.Bool
	w.OnChange(func(cfg *config.OusiaConfig) {
		called.Store(true)
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go w.Start(ctx)

	time.Sleep(50 * time.Millisecond)
	if !called.Load() {
		t.Error("expected onChange to be called")
	}
	cancel()

	// Test error reading file with a new watcher to avoid data race
	w2 := NewWatcher(filepath.Join(dir, "doesnotexist"), store, 10*time.Millisecond)
	w2.check()
}

