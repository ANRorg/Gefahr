package config

import (
	"strings"
	"testing"
	"time"
)

func validConfig() Config {
	cfg := Default()
	cfg.Pools["api"] = Pool{
		Backends: []Backend{{Name: "api-1", URL: "http://127.0.0.1:9001"}},
		Health:   Health{Path: "/health", Interval: Duration(5 * time.Second), Timeout: Duration(time.Second), UnhealthyThreshold: 2, HealthyThreshold: 1},
		Retry:    Retry{MaxAttempts: 2},
	}
	cfg.Routes = []Route{{Name: "api", Host: "example.test", PathPrefix: "/", Pool: "api", Strategy: "round_robin"}}
	return cfg
}

func TestValidateAcceptsValidConfig(t *testing.T) {
	if err := Validate(validConfig()); err != nil {
		t.Fatal(err)
	}
}

func TestValidateReportsMultipleErrors(t *testing.T) {
	cfg := validConfig()
	cfg.Routes[0].PathPrefix = "invalid"
	cfg.Routes[0].Pool = "missing"
	err := Validate(cfg)
	if err == nil || !strings.Contains(err.Error(), "path_prefix") || !strings.Contains(err.Error(), "does not exist") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateRejectsAdminCollision(t *testing.T) {
	cfg := validConfig()
	cfg.Admin.Address = cfg.Listeners[0].Address
	if err := Validate(cfg); err == nil {
		t.Fatal("expected collision error")
	}
}

func TestValidateRejectsAmbiguousRoutingAndBackendURLs(t *testing.T) {
	tests := []func(*Config){
		func(cfg *Config) { cfg.Routes[0].PathPrefix = "/public/../admin" },
		func(cfg *Config) {
			cfg.Pools["api"] = poolWithURL(cfg.Pools["api"], "http://user:secret@127.0.0.1:9001")
		},
		func(cfg *Config) { cfg.Pools["api"] = poolWithURL(cfg.Pools["api"], "http://127.0.0.1:9001/#fragment") },
	}
	for i, mutate := range tests {
		cfg := validConfig()
		mutate(&cfg)
		if err := Validate(cfg); err == nil {
			t.Fatalf("case %d was accepted", i)
		}
	}
}

func TestValidateDetectsSemanticallyDuplicateHosts(t *testing.T) {
	cfg := validConfig()
	duplicate := cfg.Routes[0]
	duplicate.Name = "duplicate"
	duplicate.Host = "EXAMPLE.TEST."
	cfg.Routes = append(cfg.Routes, duplicate)
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "duplicated") {
		t.Fatalf("error = %v", err)
	}
}

func TestValidateRejectsUnsafeMetricIdentifiers(t *testing.T) {
	cfg := validConfig()
	cfg.Routes[0].Name = "api\nforged"
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "must match") {
		t.Fatalf("error = %v", err)
	}
}

func poolWithURL(pool Pool, target string) Pool {
	pool.Backends[0].URL = target
	return pool
}
