package routing

import (
	"testing"

	"github.com/anouar/goproxy/internal/config"
)

func TestCandidatesMatchesHostCaseInsensitivelyWithoutPort(t *testing.T) {
	router := New([]config.Route{{Name: "api", Host: "API.Example.Test"}, {Name: "other", Host: "other.test"}})
	got := router.Candidates("api.example.test:8080")
	if len(got) != 1 || got[0].Name != "api" {
		t.Fatalf("candidates = %#v", got)
	}
}

func TestCandidatesIncludesCatchAll(t *testing.T) {
	router := New([]config.Route{{Name: "default"}, {Name: "api", Host: "api.test"}})
	got := router.Candidates("unknown.test")
	if len(got) != 1 || got[0].Name != "default" {
		t.Fatalf("candidates = %#v", got)
	}
}
