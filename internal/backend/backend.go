// Package backend owns concurrency-safe upstream runtime state.
package backend

import (
	"net/url"
	"sync"
	"sync/atomic"
)

// Backend represents one configured upstream and its mutable runtime state.
type Backend struct {
	name   string
	url    *url.URL
	alive  atomic.Bool
	active atomic.Int64
	mu     sync.Mutex
	fails  int
	passes int
}

// New creates an initially healthy backend.
func New(name string, target *url.URL) *Backend {
	b := &Backend{name: name, url: cloneURL(target)}
	b.alive.Store(true)
	return b
}

// Name returns the configured stable backend name.
func (b *Backend) Name() string { return b.name }

// URL returns a defensive copy of the upstream URL.
func (b *Backend) URL() *url.URL { return cloneURL(b.url) }

// Alive reports whether the backend is eligible for normal traffic.
func (b *Backend) Alive() bool { return b.alive.Load() }

// SetAlive changes eligibility and clears transition counters.
func (b *Backend) SetAlive(alive bool) {
	b.mu.Lock()
	b.fails, b.passes = 0, 0
	b.alive.Store(alive)
	b.mu.Unlock()
}

// Acquire increments the active request count and returns a release function.
// The returned function is idempotent to protect error-heavy code paths.
func (b *Backend) Acquire() func() {
	b.active.Add(1)
	var released atomic.Bool
	return func() {
		if released.CompareAndSwap(false, true) {
			b.active.Add(-1)
		}
	}
}

// Active returns the number of currently assigned requests.
func (b *Backend) Active() int64 { return b.active.Load() }

func cloneURL(u *url.URL) *url.URL {
	if u == nil {
		return nil
	}
	clone := *u
	return &clone
}
