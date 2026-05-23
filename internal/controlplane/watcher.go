package controlplane

import (
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/jaysyrk/ousia/pkg/config"
)

type UpdateFunc func(cfg *config.OusiaConfig)

type Watcher struct {
	path		string
	store		*Store
	interval	time.Duration
	lastHash	[16]byte
	onChange	[]UpdateFunc
}

func NewWatcher(path string, store *Store, interval time.Duration) *Watcher {
	return &Watcher{
		path:		path,
		store:		store,
		interval:	interval,
	}
}

func (w *Watcher) OnChange(fn UpdateFunc) {
	w.onChange = append(w.onChange, fn)
}

func (w *Watcher) Start(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.check()
		}
	}
}

func (w *Watcher) check() {
	hash, err := hashFile(w.path)
	if err != nil {
		fmt.Printf("watcher: cannot read config file: %v\n", err)
		return
	}

	if hash == w.lastHash {
		return
	}

	cfg, err := config.Load(w.path)
	if err != nil {
		fmt.Printf("watcher: invalid config, keeping current: %v\n", err)
		return
	}

	w.lastHash = hash
	w.store.Set(cfg)

	fmt.Println("watcher: config changed, applying updates")

	for _, fn := range w.onChange {
		fn(cfg)
	}
}

func hashFile(path string) ([16]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return [16]byte{}, err
	}
	defer f.Close()

	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		return [16]byte{}, err
	}

	var out [16]byte
	copy(out[:], h.Sum(nil))
	return out, nil
}
