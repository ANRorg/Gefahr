package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/anouar/goproxy/internal/config"
)

func TestReloadRejectsListenerMutationAndRetainsConfig(t *testing.T) {
	cfg, err := config.Load(strings.NewReader(testYAML))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "proxy.yaml")
	changed := strings.Replace(testYAML, "address: :8080", "address: :8081", 1)
	if err := os.WriteFile(path, []byte(changed), 0o600); err != nil {
		t.Fatal(err)
	}
	runtime, err := New(cfg, path, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := runtime.Reload(context.Background()); err == nil || !strings.Contains(err.Error(), "requires restart") {
		t.Fatalf("error = %v", err)
	}
	if runtime.Config().Listeners[0].Address != ":8080" {
		t.Fatal("rejected config was published")
	}
}

func TestImmutableCompatibleRejectsServerLevelBounds(t *testing.T) {
	current := config.Default()
	next := current
	next.Timeouts.Write++
	next.Limits.MaxHeaderBytes++
	err := immutableCompatible(current, next)
	if err == nil || !strings.Contains(err.Error(), "timeouts.write") || !strings.Contains(err.Error(), "limits.max_header_bytes") {
		t.Fatalf("error = %v", err)
	}
}

const testYAML = `
listeners:
  - address: :8080
routes:
  - name: api
    host: api.test
    path_prefix: /
    pool: api
    strategy: round_robin
pools:
  api:
    backends:
      - name: one
        url: http://127.0.0.1:9001
    health:
      path: /health
      interval: 5s
      timeout: 1s
      unhealthy_threshold: 2
      healthy_threshold: 1
    retry:
      max_attempts: 1
`
