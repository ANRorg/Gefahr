package proxy

import (
	"net"
	"net/http"
	"time"

	"github.com/anouar/goproxy/internal/config"
)

// NewTransport returns a bounded, HTTP/2-capable upstream connection pool.
func NewTransport(cfg config.Config) *http.Transport {
	base := http.DefaultTransport.(*http.Transport).Clone()
	base.Proxy = http.ProxyFromEnvironment
	base.DialContext = (&net.Dialer{Timeout: cfg.Timeouts.Dial.Value(), KeepAlive: 30 * time.Second}).DialContext
	base.ForceAttemptHTTP2 = true
	base.ResponseHeaderTimeout = cfg.Timeouts.ResponseHeader.Value()
	base.IdleConnTimeout = cfg.Timeouts.Idle.Value()
	base.MaxIdleConns = 256
	base.MaxIdleConnsPerHost = 32
	return base
}
