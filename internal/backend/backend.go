// Package backend owns concurrency-safe upstream runtime state.
package backend

import (
	"net/url"
	"sync"
	"sync/atomic"
)

// Backend represents one configured upstream and its mutable runtime state.
type Backend struct {
	name         string
	url          *url.URL
	alive        atomic.Bool
	active       atomic.Int64
	mu           sync.Mutex
	probeFails   int
	probePasses  int
	passiveFails int
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
	b.probeFails, b.probePasses, b.passiveFails = 0, 0, 0
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

// RecordProbe applies thresholded active-health evidence and returns true when
// the externally visible health state changed.
func (b *Backend) RecordProbe(success bool, healthyThreshold, unhealthyThreshold int) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	before := b.alive.Load()
	if success {
		b.probeFails = 0
		b.probePasses++
		if !before && b.probePasses >= healthyThreshold {
			b.alive.Store(true)
			b.probePasses = 0
			b.passiveFails = 0
		}
	} else {
		b.probePasses = 0
		b.probeFails++
		if before && b.probeFails >= unhealthyThreshold {
			b.alive.Store(false)
			b.probeFails = 0
		}
	}
	return before != b.alive.Load()
}

// RecordPassiveFailure removes a live backend after threshold consecutive
// real-request transport failures. Active probes remain responsible for
// bringing an ejected backend back into service.
func (b *Backend) RecordPassiveFailure(threshold int) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.alive.Load() {
		return false
	}
	b.passiveFails++
	if b.passiveFails < threshold {
		return false
	}
	b.passiveFails = 0
	b.alive.Store(false)
	return true
}

// RecordPassiveSuccess clears consecutive real-request failures.
func (b *Backend) RecordPassiveSuccess() {
	b.mu.Lock()
	b.passiveFails = 0
	b.mu.Unlock()
}

func cloneURL(u *url.URL) *url.URL {
	if u == nil {
		return nil
	}
	clone := *u
	return &clone
}
