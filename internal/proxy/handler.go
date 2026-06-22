// Package proxy composes routing, balancing, and Go's streaming reverse proxy.
package proxy

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/anouar/goproxy/internal/backend"
	"github.com/anouar/goproxy/internal/balance"
	responsecache "github.com/anouar/goproxy/internal/cache"
	"github.com/anouar/goproxy/internal/config"
	"github.com/anouar/goproxy/internal/routing"
)

type runtimePool struct {
	backends                []*backend.Backend
	balancers               map[string]balance.Balancer
	retry                   config.Retry
	passiveFailureThreshold int
}

// Handler is an immutable routing snapshot with concurrency-safe backend state.
type Handler struct {
	router       *routing.Router
	pools        map[string]*runtimePool
	transport    http.RoundTripper
	maxBodyBytes int64
	cache        *responsecache.Cache
	cacheTTL     time.Duration
	maxCacheBody int64
	observer     Observer
}

// Observer receives one bounded-cardinality record after every public request.
type Observer interface {
	ObserveRequest(requestID, route, method, path, backend string, status, attempts int, cacheResult string, duration time.Duration)
}

// Option customizes a Handler without expanding its constructor over time.
type Option func(*Handler)

// WithObserver installs request telemetry and logging observation.
func WithObserver(observer Observer) Option { return func(h *Handler) { h.observer = observer } }

// NewHandler compiles validated configuration into a request handler.
func NewHandler(cfg config.Config, transport http.RoundTripper, options ...Option) (*Handler, error) {
	if err := config.Validate(cfg); err != nil {
		return nil, err
	}
	if transport == nil {
		transport = NewTransport(cfg)
	}
	h := &Handler{router: routing.New(cfg.Routes), pools: make(map[string]*runtimePool, len(cfg.Pools)), transport: transport, maxBodyBytes: cfg.Limits.MaxBodyBytes, cache: responsecache.New(cfg.Cache.MaxEntries, cfg.Cache.MaxBytes), cacheTTL: cfg.Cache.DefaultTTL.Value(), maxCacheBody: cfg.Cache.MaxBytes}
	for name, poolCfg := range cfg.Pools {
		pool := &runtimePool{retry: poolCfg.Retry, passiveFailureThreshold: poolCfg.Health.UnhealthyThreshold, balancers: map[string]balance.Balancer{"round_robin": &balance.RoundRobin{}, "least_connections": &balance.LeastConnections{}}}
		for _, backendCfg := range poolCfg.Backends {
			target, err := url.Parse(backendCfg.URL)
			if err != nil {
				return nil, fmt.Errorf("parse backend %s: %w", backendCfg.Name, err)
			}
			pool.backends = append(pool.backends, backend.New(backendCfg.Name, target))
		}
		h.pools[name] = pool
	}
	for _, option := range options {
		option(h)
	}
	return h, nil
}

// ServeHTTP routes and streams one request through a selected healthy backend.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	started := time.Now()
	requestID := newRequestID()
	w.Header().Set("X-Request-ID", requestID)
	status := &statusWriter{ResponseWriter: w, status: http.StatusOK}
	w = status
	routeName, backendName, cacheResult, attempts := "unmatched", "", "bypass", 0
	defer func() {
		if h.observer != nil {
			h.observer.ObserveRequest(requestID, routeName, r.Method, r.URL.Path, backendName, status.status, attempts, cacheResult, time.Since(started))
		}
	}()
	if r.ContentLength > h.maxBodyBytes {
		writeError(w, http.StatusRequestEntityTooLarge, "request_too_large", "request body too large")
		return
	}
	if r.Body != nil && r.Body != http.NoBody {
		r.Body = http.MaxBytesReader(w, r.Body, h.maxBodyBytes)
	}
	route, ok := h.router.Match(r.Host, r.URL.Path)
	if !ok {
		writeError(w, http.StatusNotFound, "route_not_found", "route not found")
		return
	}
	pool := h.pools[route.Pool]
	routeName = route.Name
	cacheKey := ""
	if route.Cache.Enabled {
		if eligible, _ := responsecache.RequestEligible(r); eligible {
			cacheKey = responsecache.Key(r)
			if cached, hit := h.cache.Get(cacheKey); hit {
				cacheResult = "hit"
				writeCached(w, cached)
				return
			}
			cacheResult = "miss"
		}
	}
	selected, err := pool.balancers[route.Strategy].Next(pool.backends)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "no_healthy_upstream", "no healthy upstream")
		return
	}
	release := h.acquire(route.Pool, selected)
	backendName = selected.Name()
	attempts = 1

	target := selected.URL()
	originalHost := r.Host
	retrying := &retryTransport{base: h.transport, poolName: route.Pool, pool: pool, balancer: pool.balancers[route.Strategy], first: selected, firstRelease: release, original: r, rewriteHost: route.RewriteHost, attempts: 1, lastBackend: selected.Name(), handler: h}
	defer func() { attempts, backendName = retrying.attempts, retrying.lastBackend }()
	rp := &httputil.ReverseProxy{
		Transport: retrying,
		Rewrite: func(req *httputil.ProxyRequest) {
			req.SetURL(target)
			if !route.RewriteHost {
				req.Out.Host = originalHost
			}
			req.SetXForwarded()
			req.Out.Header.Set("Forwarded", forwardedValue(req.In))
		},
		ErrorHandler: func(w http.ResponseWriter, _ *http.Request, err error) {
			var tooLarge *http.MaxBytesError
			switch {
			case errors.As(err, &tooLarge):
				writeError(w, http.StatusRequestEntityTooLarge, "request_too_large", "request body too large")
			case errors.Is(err, context.DeadlineExceeded):
				writeError(w, http.StatusGatewayTimeout, "upstream_timeout", "upstream timed out")
			default:
				writeError(w, http.StatusBadGateway, "bad_gateway", "upstream request failed")
			}
		},
	}
	if cacheKey != "" {
		rp.ModifyResponse = func(response *http.Response) error {
			decision := responsecache.Evaluate(r, response.StatusCode, response.Header, h.cacheTTL)
			if decision.Cacheable {
				response.Body = &cacheCaptureBody{ReadCloser: response.Body, max: h.maxCacheBody, commit: func(body []byte) {
					h.cache.Set(cacheKey, responsecache.Response{Status: response.StatusCode, Header: response.Header, Body: body}, decision.TTL)
				}}
			}
			return nil
		}
	}
	rp.ServeHTTP(w, r)
}

func writeCached(w http.ResponseWriter, response responsecache.Response) {
	for name, values := range response.Header {
		for _, value := range values {
			w.Header().Add(name, value)
		}
	}
	age := int64(time.Since(response.Stored).Seconds())
	if age < 0 {
		age = 0
	}
	w.Header().Set("Age", strconv.FormatInt(age, 10))
	w.WriteHeader(response.Status)
	_, _ = w.Write(response.Body)
}

type cacheCaptureBody struct {
	io.ReadCloser
	buffer    []byte
	max       int64
	overflow  bool
	committed bool
	commit    func([]byte)
}

func (b *cacheCaptureBody) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	if !b.overflow && n > 0 {
		if int64(len(b.buffer)+n) <= b.max {
			b.buffer = append(b.buffer, p[:n]...)
		} else {
			b.overflow = true
			b.buffer = nil
		}
	}
	if err == io.EOF && !b.overflow && !b.committed {
		b.committed = true
		b.commit(append([]byte(nil), b.buffer...))
	}
	return n, err
}

type retryTransport struct {
	base         http.RoundTripper
	poolName     string
	pool         *runtimePool
	balancer     balance.Balancer
	first        *backend.Backend
	firstRelease func()
	original     *http.Request
	rewriteHost  bool
	attempts     int
	lastBackend  string
	handler      *Handler
}

func (t *retryTransport) RoundTrip(initial *http.Request) (*http.Response, error) {
	request := initial
	selected := t.first
	release := t.firstRelease
	attempts := t.pool.retry.MaxAttempts
	if !canRetry(t.original) {
		attempts = 1
	}
	for attempt := 1; attempt <= attempts; attempt++ {
		resp, err := t.base.RoundTrip(request)
		if err == nil {
			selected.RecordPassiveSuccess()
			resp.Body = &releaseBody{ReadCloser: resp.Body, release: release}
			return resp, nil
		}
		release()
		if !isBackendFailure(err) {
			return nil, err
		}
		selected.RecordPassiveFailure(t.pool.passiveFailureThreshold)
		if attempt == attempts {
			return nil, err
		}
		var selectErr error
		selected, selectErr = t.balancer.Next(t.pool.backends)
		if selectErr != nil {
			return nil, selectErr
		}
		release = t.handler.acquire(t.poolName, selected)
		t.attempts++
		t.lastBackend = selected.Name()
		request = initial.Clone(initial.Context())
		if t.original.GetBody != nil {
			request.Body, err = t.original.GetBody()
			if err != nil {
				release()
				return nil, err
			}
		}
		rewrite := &httputil.ProxyRequest{In: t.original, Out: request}
		rewrite.SetURL(selected.URL())
		if !t.rewriteHost {
			request.Host = t.original.Host
		}
	}
	return nil, errors.New("retry attempts exhausted")
}

func isBackendFailure(err error) bool {
	var tooLarge *http.MaxBytesError
	return !errors.As(err, &tooLarge) && !errors.Is(err, context.Canceled)
}

type statusWriter struct {
	http.ResponseWriter
	status int
	wrote  bool
}

func (w *statusWriter) WriteHeader(status int) {
	if w.wrote {
		return
	}
	w.wrote, w.status = true, status
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusWriter) Write(p []byte) (int, error) {
	if !w.wrote {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(p)
}

func (w *statusWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

func newRequestID() string {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 16)
	}
	return hex.EncodeToString(bytes[:])
}

func canRetry(r *http.Request) bool {
	switch r.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
		return r.Body == nil || r.Body == http.NoBody || r.GetBody != nil
	default:
		return false
	}
}

type releaseBody struct {
	io.ReadCloser
	release func()
}

func (b *releaseBody) Close() error {
	err := b.ReadCloser.Close()
	b.release()
	return err
}

func forwardedValue(r *http.Request) string {
	client := r.RemoteAddr
	if host, _, err := net.SplitHostPort(client); err == nil {
		client = host
	}
	proto := "http"
	if r.TLS != nil {
		proto = "https"
	}
	return "for=" + quoteForwarded(client) + ";host=" + quoteForwarded(r.Host) + ";proto=" + proto
}

func quoteForwarded(value string) string {
	value = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, value)
	return strconv.Quote(value)
}
