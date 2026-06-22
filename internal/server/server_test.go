package server

import (
	"net/http"
	"testing"
	"time"

	"github.com/anouar/goproxy/internal/config"
)

func TestNewPublicAppliesRequestBounds(t *testing.T) {
	cfg := config.Default()
	cfg.Timeouts.ReadHeader = config.Duration(3 * time.Second)
	cfg.Timeouts.ReadBody = config.Duration(20 * time.Second)
	cfg.Timeouts.Write = config.Duration(45 * time.Second)
	cfg.Limits.MaxHeaderBytes = 12345
	managed := NewPublic(cfg.Listeners[0], cfg, http.NotFoundHandler(), nil)
	if managed.HTTP.ReadHeaderTimeout != 3*time.Second || managed.HTTP.MaxHeaderBytes != 12345 {
		t.Fatalf("server = %#v", managed.HTTP)
	}
	if managed.HTTP.ReadTimeout != 20*time.Second || managed.HTTP.WriteTimeout != 45*time.Second {
		t.Fatal("body read and client write deadlines were not applied")
	}
}
