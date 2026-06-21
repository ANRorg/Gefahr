package config

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
)

// Store publishes immutable configuration snapshots atomically. A reload is
// visible in full or not at all, so requests never observe partial changes.
type Store struct {
	current atomic.Pointer[Config]
	reload  sync.Mutex
}

// NewStore creates a store containing initial.
func NewStore(initial Config) *Store {
	s := &Store{}
	s.current.Store(&initial)
	return s
}

// Current returns the active immutable configuration snapshot.
func (s *Store) Current() *Config { return s.current.Load() }

// Reload loads and validates path before atomically publishing it. The active
// configuration remains unchanged when loading fails.
func (s *Store) Reload(path string) error {
	s.reload.Lock()
	defer s.reload.Unlock()
	next, err := LoadFile(path)
	if err != nil {
		return fmt.Errorf("reload: %w", err)
	}
	s.current.Store(&next)
	return nil
}

// Watch reloads path for every signal received until ctx is canceled.
func (s *Store) Watch(ctx context.Context, path string, signals <-chan struct{}, onError func(error)) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-signals:
			if err := s.Reload(path); err != nil && onError != nil {
				onError(err)
			}
		}
	}
}
