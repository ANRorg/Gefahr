// Package config defines GoProxy's declarative configuration and defaults.
package config

import (
	"fmt"
	"time"

	"gopkg.in/yaml.v3"
)

// Duration wraps time.Duration so configuration can use values such as "5s".
type Duration time.Duration

// UnmarshalYAML parses a human-readable duration such as "250ms" or "5s".
func (d *Duration) UnmarshalYAML(node *yaml.Node) error {
	parsed, err := time.ParseDuration(node.Value)
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", node.Value, err)
	}
	*d = Duration(parsed)
	return nil
}

// Value returns the standard-library duration value.
func (d Duration) Value() time.Duration { return time.Duration(d) }

// Config is the complete startup and reloadable configuration.
type Config struct {
	Listeners []Listener      `yaml:"listeners"`
	Admin     Admin           `yaml:"admin"`
	Routes    []Route         `yaml:"routes"`
	Pools     map[string]Pool `yaml:"pools"`
	Timeouts  Timeouts        `yaml:"timeouts"`
	Limits    Limits          `yaml:"limits"`
	Cache     Cache           `yaml:"cache"`
	Logging   Logging         `yaml:"logging"`
}

// Listener describes one public HTTP or HTTPS listener.
type Listener struct {
	Address string     `yaml:"address"`
	TLS     *TLSConfig `yaml:"tls,omitempty"`
}

// TLSConfig identifies a PEM certificate and key for a listener.
type TLSConfig struct {
	CertFile string `yaml:"cert_file"`
	KeyFile  string `yaml:"key_file"`
}

// Admin configures the private operational HTTP listener.
type Admin struct {
	Address string `yaml:"address"`
}

// Route maps an incoming host and path prefix to a backend pool.
type Route struct {
	Name        string     `yaml:"name"`
	Host        string     `yaml:"host"`
	PathPrefix  string     `yaml:"path_prefix"`
	Pool        string     `yaml:"pool"`
	Strategy    string     `yaml:"strategy"`
	RewriteHost bool       `yaml:"rewrite_host"`
	Cache       RouteCache `yaml:"cache"`
}

// RouteCache controls response caching for a route.
type RouteCache struct {
	Enabled bool `yaml:"enabled"`
}

// Pool is a group of interchangeable upstream servers.
type Pool struct {
	Backends []Backend `yaml:"backends"`
	Health   Health    `yaml:"health"`
	Retry    Retry     `yaml:"retry"`
}

// Backend identifies one upstream server.
type Backend struct {
	Name string `yaml:"name"`
	URL  string `yaml:"url"`
}

// Health controls active backend probes and transition thresholds.
type Health struct {
	Path               string   `yaml:"path"`
	Interval           Duration `yaml:"interval"`
	Timeout            Duration `yaml:"timeout"`
	UnhealthyThreshold int      `yaml:"unhealthy_threshold"`
	HealthyThreshold   int      `yaml:"healthy_threshold"`
}

// Retry controls attempts made before an upstream response is committed.
type Retry struct {
	MaxAttempts int `yaml:"max_attempts"`
}

// Timeouts bounds public and upstream network operations.
type Timeouts struct {
	ReadHeader     Duration `yaml:"read_header"`
	Idle           Duration `yaml:"idle"`
	Shutdown       Duration `yaml:"shutdown"`
	Dial           Duration `yaml:"dial"`
	ResponseHeader Duration `yaml:"response_header"`
}

// Limits bounds request metadata and body sizes.
type Limits struct {
	MaxHeaderBytes int   `yaml:"max_header_bytes"`
	MaxBodyBytes   int64 `yaml:"max_body_bytes"`
}

// Cache controls the process-wide bounded response cache.
type Cache struct {
	MaxEntries int      `yaml:"max_entries"`
	MaxBytes   int64    `yaml:"max_bytes"`
	DefaultTTL Duration `yaml:"default_ttl"`
}

// Logging controls structured application logging.
type Logging struct {
	Level string `yaml:"level"`
}

// Default returns conservative production-minded defaults.
func Default() Config {
	return Config{
		Listeners: []Listener{{Address: ":8080"}},
		Admin:     Admin{Address: "127.0.0.1:9090"},
		Pools:     make(map[string]Pool),
		Timeouts:  Timeouts{ReadHeader: Duration(10 * time.Second), Idle: Duration(60 * time.Second), Shutdown: Duration(30 * time.Second), Dial: Duration(5 * time.Second), ResponseHeader: Duration(30 * time.Second)},
		Limits:    Limits{MaxHeaderBytes: 16 << 10, MaxBodyBytes: 10 << 20},
		Cache:     Cache{MaxEntries: 1000, MaxBytes: 64 << 20, DefaultTTL: Duration(30 * time.Second)},
		Logging:   Logging{Level: "info"},
	}
}
