package proxy

import (
	"testing"
	"time"

	"github.com/anouar/goproxy/internal/config"
)

func TestNewTransportAppliesTimeoutsAndBounds(t *testing.T) {
	cfg := proxyConfig()
	cfg.Timeouts.Dial = config.Duration(3 * time.Second)
	cfg.Timeouts.ResponseHeader = config.Duration(7 * time.Second)
	transport := NewTransport(cfg)
	if transport.ResponseHeaderTimeout != 7*time.Second {
		t.Fatalf("response header timeout = %s", transport.ResponseHeaderTimeout)
	}
	if transport.MaxIdleConns <= 0 || transport.MaxIdleConnsPerHost <= 0 {
		t.Fatal("idle connection pool is unbounded")
	}
	if !transport.ForceAttemptHTTP2 {
		t.Fatal("HTTP/2 is disabled")
	}
}
