package config

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// Validate reports all configuration errors that can be checked without
// opening listeners or contacting upstreams.
func Validate(cfg Config) error {
	var errs []error
	if len(cfg.Listeners) == 0 {
		errs = append(errs, errors.New("at least one listener is required"))
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
	}

	seenNames, seenMatches := map[string]bool{}, map[string]bool{}
	for i, route := range cfg.Routes {
		field := fmt.Sprintf("routes[%d]", i)
		if route.Name == "" {
			errs = append(errs, fmt.Errorf("%s.name is required", field))
		}
		if seenNames[route.Name] {
			errs = append(errs, fmt.Errorf("route name %q is duplicated", route.Name))
		}
		seenNames[route.Name] = true
		if route.PathPrefix == "" || !strings.HasPrefix(route.PathPrefix, "/") {
			errs = append(errs, fmt.Errorf("%s.path_prefix must start with /", field))
		}
		match := strings.ToLower(route.Host) + "\x00" + route.PathPrefix
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

	for name, pool := range cfg.Pools {
		if len(pool.Backends) == 0 {
			errs = append(errs, fmt.Errorf("pool %q requires at least one backend", name))
		}
		seenBackends := map[string]bool{}
		for i, backend := range pool.Backends {
			u, err := url.Parse(backend.URL)
			if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
				errs = append(errs, fmt.Errorf("pools.%s.backends[%d].url must be an absolute http(s) URL", name, i))
			}
			if seenBackends[backend.Name] || backend.Name == "" {
				errs = append(errs, fmt.Errorf("pools.%s backend names must be non-empty and unique", name))
			}
			seenBackends[backend.Name] = true
		}
		if pool.Health.Path == "" || !strings.HasPrefix(pool.Health.Path, "/") {
			errs = append(errs, fmt.Errorf("pools.%s.health.path must start with /", name))
		}
		if pool.Health.Interval <= 0 || pool.Health.Timeout <= 0 || pool.Health.UnhealthyThreshold <= 0 || pool.Health.HealthyThreshold <= 0 {
			errs = append(errs, fmt.Errorf("pools.%s health values must be positive", name))
		}
		if pool.Retry.MaxAttempts < 1 || pool.Retry.MaxAttempts > 2 {
			errs = append(errs, fmt.Errorf("pools.%s.retry.max_attempts must be 1 or 2", name))
		}
	}
	if cfg.Timeouts.ReadHeader <= 0 || cfg.Timeouts.Idle <= 0 || cfg.Timeouts.Shutdown <= 0 || cfg.Timeouts.Dial <= 0 || cfg.Timeouts.ResponseHeader <= 0 {
		errs = append(errs, errors.New("all timeouts must be positive"))
	}
	if cfg.Limits.MaxHeaderBytes <= 0 || cfg.Limits.MaxBodyBytes <= 0 {
		errs = append(errs, errors.New("all request limits must be positive"))
	}
	if cfg.Cache.MaxEntries <= 0 || cfg.Cache.MaxBytes <= 0 || cfg.Cache.DefaultTTL <= 0 {
		errs = append(errs, errors.New("all cache bounds must be positive"))
	}
	if cfg.Logging.Level != "debug" && cfg.Logging.Level != "info" && cfg.Logging.Level != "warn" && cfg.Logging.Level != "error" {
		errs = append(errs, errors.New("logging.level must be debug, info, warn, or error"))
	}
	return errors.Join(errs...)
}
