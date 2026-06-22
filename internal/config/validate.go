package config

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strings"
)

var identifierPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{0,127}$`)

// Validate reports all configuration errors that can be checked without
// opening listeners or contacting upstreams.
func Validate(cfg Config) error {
	var errs []error
	const (
		maxListeners = 16
		maxRoutes    = 1000
		maxPools     = 1000
		maxBackends  = 10000
	)
	if len(cfg.Listeners) == 0 {
		errs = append(errs, errors.New("at least one listener is required"))
	} else if len(cfg.Listeners) > maxListeners {
		errs = append(errs, fmt.Errorf("listener count exceeds %d", maxListeners))
	}
	seenListeners := map[string]bool{}
	for i, listener := range cfg.Listeners {
		if listener.Address == "" {
			errs = append(errs, fmt.Errorf("listeners[%d].address is required", i))
		}
		if seenListeners[listener.Address] {
			errs = append(errs, fmt.Errorf("listener address %q is duplicated", listener.Address))
		}
		seenListeners[listener.Address] = true
		if listener.TLS != nil && (listener.TLS.CertFile == "" || listener.TLS.KeyFile == "") {
			errs = append(errs, fmt.Errorf("listeners[%d].tls requires cert_file and key_file", i))
		}
	}
	if cfg.Admin.Address == "" {
		errs = append(errs, errors.New("admin.address is required"))
	}
	if seenListeners[cfg.Admin.Address] {
		errs = append(errs, errors.New("admin address must differ from public listeners"))
	}
	if len(cfg.Routes) == 0 {
		errs = append(errs, errors.New("at least one route is required"))
	} else if len(cfg.Routes) > maxRoutes {
		errs = append(errs, fmt.Errorf("route count exceeds %d", maxRoutes))
	}
	if len(cfg.Pools) > maxPools {
		errs = append(errs, fmt.Errorf("pool count exceeds %d", maxPools))
	}

	seenNames, seenMatches := map[string]bool{}, map[string]bool{}
	for i, route := range cfg.Routes {
		field := fmt.Sprintf("routes[%d]", i)
		if !identifierPattern.MatchString(route.Name) {
			errs = append(errs, fmt.Errorf("%s.name must match %s", field, identifierPattern))
		}
		if seenNames[route.Name] {
			errs = append(errs, fmt.Errorf("route name %q is duplicated", route.Name))
		}
		seenNames[route.Name] = true
		if route.Host != strings.TrimSpace(route.Host) {
			errs = append(errs, fmt.Errorf("%s.host must not contain surrounding whitespace", field))
		}
		if route.PathPrefix == "" || !strings.HasPrefix(route.PathPrefix, "/") {
			errs = append(errs, fmt.Errorf("%s.path_prefix must start with /", field))
		} else if ambiguousPath(route.PathPrefix) {
			errs = append(errs, fmt.Errorf("%s.path_prefix contains ambiguous separators or segments", field))
		}
		match := normalizedHost(route.Host) + "\x00" + route.PathPrefix
		if seenMatches[match] {
			errs = append(errs, fmt.Errorf("route match host=%q path_prefix=%q is duplicated", route.Host, route.PathPrefix))
		}
		seenMatches[match] = true
		if _, ok := cfg.Pools[route.Pool]; !ok {
			errs = append(errs, fmt.Errorf("%s.pool %q does not exist", field, route.Pool))
		}
		if route.Strategy != "round_robin" && route.Strategy != "least_connections" {
			errs = append(errs, fmt.Errorf("%s.strategy must be round_robin or least_connections", field))
		}
	}

	totalBackends := 0
	for name, pool := range cfg.Pools {
		if !identifierPattern.MatchString(name) {
			errs = append(errs, fmt.Errorf("pool name %q must match %s", name, identifierPattern))
		}
		totalBackends += len(pool.Backends)
		if len(pool.Backends) == 0 {
			errs = append(errs, fmt.Errorf("pool %q requires at least one backend", name))
		}
		seenBackends := map[string]bool{}
		for i, backend := range pool.Backends {
			u, err := url.Parse(backend.URL)
			if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.Fragment != "" {
				errs = append(errs, fmt.Errorf("pools.%s.backends[%d].url must be an absolute http(s) URL", name, i))
			}
			if seenBackends[backend.Name] || !identifierPattern.MatchString(backend.Name) {
				errs = append(errs, fmt.Errorf("pools.%s backend names must match %s and be unique", name, identifierPattern))
			}
			seenBackends[backend.Name] = true
		}
		if pool.Health.Path == "" || !strings.HasPrefix(pool.Health.Path, "/") || ambiguousPath(pool.Health.Path) || strings.ContainsAny(pool.Health.Path, "?#") {
			errs = append(errs, fmt.Errorf("pools.%s.health.path must start with /", name))
		}
		if pool.Health.Interval <= 0 || pool.Health.Timeout <= 0 || pool.Health.UnhealthyThreshold <= 0 || pool.Health.HealthyThreshold <= 0 {
			errs = append(errs, fmt.Errorf("pools.%s health values must be positive", name))
		}
		if pool.Retry.MaxAttempts < 1 || pool.Retry.MaxAttempts > 2 {
			errs = append(errs, fmt.Errorf("pools.%s.retry.max_attempts must be 1 or 2", name))
		}
	}
	if totalBackends > maxBackends {
		errs = append(errs, fmt.Errorf("backend count exceeds %d", maxBackends))
	}
	if cfg.Timeouts.ReadHeader <= 0 || cfg.Timeouts.ReadBody <= 0 || cfg.Timeouts.Write <= 0 || cfg.Timeouts.Idle <= 0 || cfg.Timeouts.Shutdown <= 0 || cfg.Timeouts.Dial <= 0 || cfg.Timeouts.ResponseHeader <= 0 {
		errs = append(errs, errors.New("all timeouts must be positive"))
	}
	if cfg.Limits.MaxHeaderBytes <= 0 || cfg.Limits.MaxBodyBytes <= 0 || cfg.Limits.MaxConcurrentRequests <= 0 || cfg.Limits.MaxConnections <= 0 {
		errs = append(errs, errors.New("all request limits must be positive"))
	}
	if cfg.Limits.MaxConcurrentRequests > 100000 {
		errs = append(errs, errors.New("limits.max_concurrent_requests must not exceed 100000"))
	}
	if cfg.Limits.MaxConnections > 1000000 {
		errs = append(errs, errors.New("limits.max_connections must not exceed 1000000"))
	}
	if cfg.Cache.MaxEntries <= 0 || cfg.Cache.MaxBytes <= 0 || cfg.Cache.DefaultTTL <= 0 {
		errs = append(errs, errors.New("all cache bounds must be positive"))
	}
	if cfg.Logging.Level != "debug" && cfg.Logging.Level != "info" && cfg.Logging.Level != "warn" && cfg.Logging.Level != "error" {
		errs = append(errs, errors.New("logging.level must be debug, info, warn, or error"))
	}
	return errors.Join(errs...)
}

func normalizedHost(host string) string {
	host = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(host)), ".")
	if parsed, _, err := net.SplitHostPort(host); err == nil {
		return parsed
	}
	return host
}

func ambiguousPath(value string) bool {
	lower := strings.ToLower(value)
	if strings.Contains(lower, "%2f") || strings.Contains(lower, "%5c") || strings.Contains(lower, "%25") || strings.Contains(value, `\`) {
		return true
	}
	for _, segment := range strings.Split(value, "/") {
		if segment == "." || segment == ".." || strings.EqualFold(segment, "%2e") || strings.EqualFold(segment, "%2e%2e") {
			return true
		}
	}
	return false
}
