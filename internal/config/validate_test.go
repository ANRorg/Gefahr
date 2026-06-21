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
