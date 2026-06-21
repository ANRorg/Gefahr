package backend

import (
	"net/url"
	"testing"
)

func TestAcquireReleaseIsIdempotent(t *testing.T) {
	b := New("one", &url.URL{Scheme: "http", Host: "example.test"})
	release := b.Acquire()
	if b.Active() != 1 {
		t.Fatalf("active = %d", b.Active())
	}
	release()
	release()
	if b.Active() != 0 {
		t.Fatalf("active = %d", b.Active())
	}
}

func TestURLReturnsCopy(t *testing.T) {
	b := New("one", &url.URL{Scheme: "http", Host: "example.test"})
	u := b.URL()
	u.Host = "changed.test"
	if b.URL().Host != "example.test" {
		t.Fatal("backend URL was mutated")
	}
}

func TestPassiveFailureEjectsAtThresholdAndSuccessResets(t *testing.T) {
	b := New("one", &url.URL{Scheme: "http", Host: "example.test"})
	if b.RecordPassiveFailure(2) {
		t.Fatal("ejected too early")
	}
	b.RecordPassiveSuccess()
	if b.RecordPassiveFailure(2) {
		t.Fatal("success did not reset failures")
	}
	if !b.RecordPassiveFailure(2) || b.Alive() {
		t.Fatal("backend was not ejected")
	}
}
