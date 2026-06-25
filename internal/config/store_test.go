package config

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
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

func TestStoreWatchReloadsAndReportsErrors(t *testing.T) {
	initial := validConfig()
	initial.Routes[0].Name = "original"
	store := NewStore(initial)
	path := filepath.Join(t.TempDir(), "proxy.yaml")
	if err := os.WriteFile(path, []byte(validYAML), 0o600); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	signals := make(chan struct{})
	errs := make(chan error, 1)
	done := make(chan struct{})
	go func() {
		store.Watch(ctx, path, signals, func(err error) { errs <- err })
		close(done)
	}()

	signals <- struct{}{}
	waitForStoreRoute(t, store, "api")

	if err := os.WriteFile(path, []byte("unknown: true\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	signals <- struct{}{}
	select {
	case err := <-errs:
		if err == nil {
			t.Fatal("nil reload error")
		}
	case <-time.After(time.Second):
		t.Fatal("watch did not report reload error")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("watch did not stop")
	}
}

func waitForStoreRoute(t *testing.T, store *Store, name string) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if store.Current().Routes[0].Name == name {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("route did not become %q", name)
}
