package observability

import "time"

// RequestObserver receives completed public request records.
type RequestObserver interface {
	ObserveRequest(requestID, route, method, path, backend string, status, attempts int, cacheResult string, duration time.Duration)
}

// BackendObserver receives backend gauge updates.
type BackendObserver interface {
	SetBackendHealth(pool, backend string, healthy bool)
	SetBackendActive(pool, backend string, active int64)
}

// Fanout sends observations to independent logging and metrics consumers.
type Fanout struct {
	Requests []RequestObserver
	Backends []BackendObserver
}

// ObserveRequest forwards a completed request to every request observer.
func (f Fanout) ObserveRequest(requestID, route, method, path, backend string, status, attempts int, cacheResult string, duration time.Duration) {
	for _, observer := range f.Requests {
		observer.ObserveRequest(requestID, route, method, path, backend, status, attempts, cacheResult, duration)
	}
}

// SetBackendHealth forwards backend health state.
func (f Fanout) SetBackendHealth(pool, backend string, healthy bool) {
	for _, observer := range f.Backends {
		observer.SetBackendHealth(pool, backend, healthy)
	}
}

// SetBackendActive forwards backend active request state.
func (f Fanout) SetBackendActive(pool, backend string, active int64) {
	for _, observer := range f.Backends {
		observer.SetBackendActive(pool, backend, active)
	}
}
