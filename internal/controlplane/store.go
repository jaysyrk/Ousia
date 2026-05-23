package controlplane

import (
	"sync"

	"github.com/jaysyrk/ousia/pkg/config"
)

type Store struct {
	mu  sync.RWMutex
	cfg *config.OusiaConfig
}

func NewStore(initial *config.OusiaConfig) *Store {
	return &Store{cfg: initial}
}

func (s *Store) Get() *config.OusiaConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg
}

func (s *Store) Set(cfg *config.OusiaConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cfg = cfg
}
