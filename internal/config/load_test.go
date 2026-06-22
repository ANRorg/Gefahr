package config

import (
	"strings"
	"testing"
	"time"
)

const validYAML = `
routes:
  - name: api
    host: example.test
    path_prefix: /
    pool: api
    strategy: round_robin
pools:
  api:
    backends:
      - name: api-1
        url: http://127.0.0.1:9001
    health:
      path: /health
      interval: 5s
      timeout: 1s
      unhealthy_threshold: 2
      healthy_threshold: 1
    retry:
      max_attempts: 2
`

func TestLoadAppliesDefaultsAndDurations(t *testing.T) {
	cfg, err := Load(strings.NewReader(validYAML))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Pools["api"].Health.Interval.Value() != 5*time.Second {
		t.Fatalf("interval = %s", cfg.Pools["api"].Health.Interval.Value())
	}
	if cfg.Limits.MaxBodyBytes != 10<<20 {
		t.Fatalf("default max body = %d", cfg.Limits.MaxBodyBytes)
	}
}

func TestLoadRejectsUnknownFields(t *testing.T) {
	_, err := Load(strings.NewReader(validYAML + "unknown: true\n"))
	if err == nil || !strings.Contains(err.Error(), "field unknown not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadRejectsInvalidDuration(t *testing.T) {
	_, err := Load(strings.NewReader(strings.Replace(validYAML, "interval: 5s", "interval: soon", 1)))
	if err == nil || !strings.Contains(err.Error(), "invalid duration") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadRejectsTrailingYAMLDocument(t *testing.T) {
	_, err := Load(strings.NewReader(validYAML + "\n---\nlogging:\n  level: debug\n"))
	if err == nil || !strings.Contains(err.Error(), "multiple YAML documents") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadRejectsOversizedInput(t *testing.T) {
	_, err := Load(strings.NewReader(strings.Repeat("#", (4<<20)+1)))
	if err == nil || !strings.Contains(err.Error(), "size exceeds") {
		t.Fatalf("unexpected error: %v", err)
	}
}
