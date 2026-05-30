package controlplane

import (
	"context"
	"os"
	"path/filepath"
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

	called := false
	w.OnChange(func(cfg *config.OusiaConfig) {
		called = true
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go w.Start(ctx)

	// Trigger initial check
	time.Sleep(50 * time.Millisecond)
	if !called {
		t.Error("expected onChange to be called")
	}
	
	// Test error reading file
	w.path = filepath.Join(dir, "doesnotexist")
	w.check()
}
