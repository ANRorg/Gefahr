// Package metrics defines GoProxy's bounded-label Prometheus text contract.
package metrics

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type requestKey struct {
	route, method string
	status        int
}
type cacheKey struct{ route, result string }
type backendKey struct{ pool, backend string }

// Metrics stores counters and gauges whose labels come only from validated
// configuration or bounded enums, preventing attacker-controlled cardinality.
type Metrics struct {
	mu            sync.Mutex
	requests      map[requestKey]uint64
	durationCount map[string]uint64
	durationSum   map[string]float64
	cache         map[cacheKey]uint64
	retries       map[string]uint64
	health        map[backendKey]float64
	active        map[backendKey]int64
}

// New creates an empty metrics registry.
func New() *Metrics {
	return &Metrics{requests: map[requestKey]uint64{}, durationCount: map[string]uint64{}, durationSum: map[string]float64{}, cache: map[cacheKey]uint64{}, retries: map[string]uint64{}, health: map[backendKey]float64{}, active: map[backendKey]int64{}}
}

// Handler exposes metrics in the Prometheus text exposition format.
func (m *Metrics) Handler() http.Handler { return http.HandlerFunc(m.serveHTTP) }

// ObserveRequest records one completed public request.
func (m *Metrics) ObserveRequest(_, route, method, _, _ string, status, attempts int, cacheResult string, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.requests[requestKey{route, method, status}]++
	m.durationCount[route]++
	m.durationSum[route] += duration.Seconds()
	if cacheResult != "" {
		m.cache[cacheKey{route, cacheResult}]++
	}
	if attempts > 1 {
		m.retries[route] += uint64(attempts - 1)
	}
}

// SetBackendHealth updates the backend eligibility gauge.
func (m *Metrics) SetBackendHealth(pool, backend string, healthy bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if healthy {
		m.health[backendKey{pool, backend}] = 1
	} else {
		m.health[backendKey{pool, backend}] = 0
	}
}

// SetBackendActive updates the backend active-request gauge.
func (m *Metrics) SetBackendActive(pool, backend string, active int64) {
	m.mu.Lock()
	m.active[backendKey{pool, backend}] = active
	m.mu.Unlock()
}

func (m *Metrics) serveHTTP(w http.ResponseWriter, _ *http.Request) {
	m.mu.Lock()
	lines := m.lines()
	m.mu.Unlock()
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	_, _ = w.Write([]byte(strings.Join(lines, "\n") + "\n"))
}

func (m *Metrics) lines() []string {
	lines := []string{
		"# HELP goproxy_requests_total Completed public requests.", "# TYPE goproxy_requests_total counter",
		"# HELP goproxy_request_duration_seconds Public request duration.", "# TYPE goproxy_request_duration_seconds summary",
		"# HELP goproxy_cache_requests_total Cache outcomes for public requests.", "# TYPE goproxy_cache_requests_total counter",
		"# HELP goproxy_retries_total Upstream retry attempts.", "# TYPE goproxy_retries_total counter",
		"# HELP goproxy_backend_healthy Whether a backend is eligible for traffic.", "# TYPE goproxy_backend_healthy gauge",
		"# HELP goproxy_backend_active_requests Requests currently assigned to a backend.", "# TYPE goproxy_backend_active_requests gauge",
	}
	for key, value := range m.requests {
		lines = append(lines, fmt.Sprintf("goproxy_requests_total{method=%s,route=%s,status=%s} %d", quote(key.method), quote(key.route), quote(strconv.Itoa(key.status)), value))
	}
	for route, value := range m.durationCount {
		lines = append(lines, fmt.Sprintf("goproxy_request_duration_seconds_count{route=%s} %d", quote(route), value), fmt.Sprintf("goproxy_request_duration_seconds_sum{route=%s} %g", quote(route), m.durationSum[route]))
	}
	for key, value := range m.cache {
		lines = append(lines, fmt.Sprintf("goproxy_cache_requests_total{result=%s,route=%s} %d", quote(key.result), quote(key.route), value))
	}
	for route, value := range m.retries {
		lines = append(lines, fmt.Sprintf("goproxy_retries_total{route=%s} %d", quote(route), value))
	}
	for key, value := range m.health {
		lines = append(lines, fmt.Sprintf("goproxy_backend_healthy{backend=%s,pool=%s} %g", quote(key.backend), quote(key.pool), value))
	}
	for key, value := range m.active {
		lines = append(lines, fmt.Sprintf("goproxy_backend_active_requests{backend=%s,pool=%s} %d", quote(key.backend), quote(key.pool), value))
	}
	sort.Strings(lines[12:])
	return lines
}

func quote(value string) string { return strconv.Quote(value) }
