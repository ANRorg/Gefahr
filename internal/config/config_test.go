package config

import (
	"testing"
	"time"
)

func TestDefaultIsBounded(t *testing.T) {
	cfg := Default()
	if cfg.Limits.MaxHeaderBytes <= 0 || cfg.Limits.MaxBodyBytes <= 0 {
		t.Fatal("request limits must be positive")
	}
	if cfg.Cache.MaxEntries <= 0 || cfg.Cache.MaxBytes <= 0 {
		t.Fatal("cache limits must be positive")
	}
	if cfg.Timeouts.Shutdown.Value() != 30*time.Second {
		t.Fatalf("shutdown timeout = %s", cfg.Timeouts.Shutdown.Value())
	}
}
