package server

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/anouar/goproxy/internal/config"
)

// Managed describes one HTTP server and its optional TLS policy.
type Managed struct {
	HTTP *http.Server
	TLS  *tls.Config
}

// NewPublic creates a bounded public HTTP server.
func NewPublic(listener config.Listener, cfg config.Config, handler http.Handler, tlsConfig *tls.Config) Managed {
	return Managed{HTTP: &http.Server{Addr: listener.Address, Handler: handler, ReadHeaderTimeout: cfg.Timeouts.ReadHeader.Value(), IdleTimeout: cfg.Timeouts.Idle.Value(), WriteTimeout: 0, MaxHeaderBytes: cfg.Limits.MaxHeaderBytes}, TLS: tlsConfig}
}

// NewAdmin creates a loopback-oriented operational HTTP server.
func NewAdmin(cfg config.Config, handler http.Handler) Managed {
	return Managed{HTTP: &http.Server{Addr: cfg.Admin.Address, Handler: handler, ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 30 * time.Second, MaxHeaderBytes: 8 << 10}}
}

// Run opens every listener before serving and drains all servers when ctx is
// canceled. A bind or serve failure shuts down the complete group.
func Run(ctx context.Context, servers []Managed, shutdownTimeout time.Duration) error {
	listeners := make([]net.Listener, 0, len(servers))
	for _, managed := range servers {
		listener, err := net.Listen("tcp", managed.HTTP.Addr)
		if err != nil {
			for _, open := range listeners {
				open.Close()
			}
			return fmt.Errorf("listen on %s: %w", managed.HTTP.Addr, err)
		}
		if managed.TLS != nil {
			listener = tls.NewListener(listener, managed.TLS)
		}
		listeners = append(listeners, listener)
	}

	errCh := make(chan error, len(servers))
	var wg sync.WaitGroup
	for i, managed := range servers {
		wg.Add(1)
		go func(managed Managed, listener net.Listener) {
			defer wg.Done()
			if err := managed.HTTP.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
				errCh <- err
			}
		}(managed, listeners[i])
	}

	var runErr error
	select {
	case <-ctx.Done():
	case runErr = <-errCh:
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	for _, managed := range servers {
		_ = managed.HTTP.Shutdown(shutdownCtx)
	}
	wg.Wait()
	return runErr
}
