// Package routing selects a configured route for an incoming request.
package routing

import (
	"net"
	"strings"

	"github.com/anouar/goproxy/internal/config"
)

// Router is an immutable host-indexed route table.
type Router struct{ routes []config.Route }

// New builds an immutable route table from validated routes.
func New(routes []config.Route) *Router {
	cloned := append([]config.Route(nil), routes...)
	return &Router{routes: cloned}
}

// Candidates returns routes whose host is either empty or exactly matches
// host. An empty configured host is the explicit catch-all virtual host.
func (r *Router) Candidates(host string) []config.Route {
	host = normalizeHost(host)
	result := make([]config.Route, 0, len(r.routes))
	for _, route := range r.routes {
		if route.Host == "" || normalizeHost(route.Host) == host {
			result = append(result, route)
		}
	}
	return result
}

func normalizeHost(host string) string {
	if parsed, _, err := net.SplitHostPort(host); err == nil {
		host = parsed
	}
	return strings.TrimSuffix(strings.ToLower(strings.TrimSpace(host)), ".")
}
