// Package proxy composes routing, balancing, and Go's streaming reverse proxy.
package proxy

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/anouar/goproxy/internal/backend"
	"github.com/anouar/goproxy/internal/balance"
	"github.com/anouar/goproxy/internal/config"
	"github.com/anouar/goproxy/internal/routing"
)

type runtimePool struct {
	backends  []*backend.Backend
	balancers map[string]balance.Balancer
	retry     config.Retry
}

// Handler is an immutable routing snapshot with concurrency-safe backend state.
type Handler struct {
	router    *routing.Router
	pools     map[string]*runtimePool
	transport http.RoundTripper
}

// NewHandler compiles validated configuration into a request handler.
func NewHandler(cfg config.Config, transport http.RoundTripper) (*Handler, error) {
	if err := config.Validate(cfg); err != nil {
		return nil, err
	}
	if transport == nil {
		transport = http.DefaultTransport
	}
	h := &Handler{router: routing.New(cfg.Routes), pools: make(map[string]*runtimePool, len(cfg.Pools)), transport: transport}
	for name, poolCfg := range cfg.Pools {
		pool := &runtimePool{retry: poolCfg.Retry, balancers: map[string]balance.Balancer{"round_robin": &balance.RoundRobin{}, "least_connections": &balance.LeastConnections{}}}
		for _, backendCfg := range poolCfg.Backends {
			target, err := url.Parse(backendCfg.URL)
			if err != nil {
				return nil, fmt.Errorf("parse backend %s: %w", backendCfg.Name, err)
			}
			pool.backends = append(pool.backends, backend.New(backendCfg.Name, target))
		}
		h.pools[name] = pool
	}
	return h, nil
}

// ServeHTTP routes and streams one request through a selected healthy backend.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	route, ok := h.router.Match(r.Host, r.URL.Path)
	if !ok {
		http.Error(w, "route not found", http.StatusNotFound)
		return
	}
	pool := h.pools[route.Pool]
	selected, err := pool.balancers[route.Strategy].Next(pool.backends)
	if err != nil {
		http.Error(w, "no healthy upstream", http.StatusServiceUnavailable)
		return
	}
	release := selected.Acquire()
	defer release()

	target := selected.URL()
	originalHost := r.Host
	rp := &httputil.ReverseProxy{
		Transport: h.transport,
		Rewrite: func(req *httputil.ProxyRequest) {
			req.SetURL(target)
			if !route.RewriteHost {
				req.Out.Host = originalHost
			}
		},
		ErrorHandler: func(w http.ResponseWriter, _ *http.Request, _ error) {
			http.Error(w, "bad gateway", http.StatusBadGateway)
		},
	}
	rp.ServeHTTP(w, r)
}
