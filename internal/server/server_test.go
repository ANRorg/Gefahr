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
	cfg.Limits.MaxHeaderBytes = 12345
	managed := NewPublic(cfg.Listeners[0], cfg, http.NotFoundHandler(), nil)
	if managed.HTTP.ReadHeaderTimeout != 3*time.Second || managed.HTTP.MaxHeaderBytes != 12345 {
		t.Fatalf("server = %#v", managed.HTTP)
	}
	if managed.HTTP.WriteTimeout != 0 {
		t.Fatal("streaming responses must not have a whole-response write timeout")
	}
}
