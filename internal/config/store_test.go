package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStoreReloadPublishesValidConfig(t *testing.T) {
	store := NewStore(validConfig())
	path := filepath.Join(t.TempDir(), "proxy.yaml")
	if err := os.WriteFile(path, []byte(validYAML), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := store.Reload(path); err != nil {
		t.Fatal(err)
	}
	if store.Current().Routes[0].Name != "api" {
		t.Fatalf("route = %q", store.Current().Routes[0].Name)
	}
}

func TestStoreReloadRetainsPreviousConfigOnError(t *testing.T) {
	initial := validConfig()
	initial.Routes[0].Name = "original"
	store := NewStore(initial)
	path := filepath.Join(t.TempDir(), "proxy.yaml")
	if err := os.WriteFile(path, []byte("unknown: true\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := store.Reload(path); err == nil {
		t.Fatal("expected reload error")
	}
	if store.Current().Routes[0].Name != "original" {
		t.Fatal("invalid config was published")
	}
}
