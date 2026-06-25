// Package admin exposes operational endpoints on a separate listener.
package admin

import (
	"crypto/subtle"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

type options struct {
	auditLogger *slog.Logger
}

// Option customizes the operational handler.
type Option func(*options)

// WithAuditLogger emits one structured event for every admin request.
func WithAuditLogger(logger *slog.Logger) Option {
	return func(options *options) {
		options.auditLogger = logger
	}
}

// NewHandler builds liveness, readiness, and optional metrics routes.
func NewHandler(live, ready func() bool, metrics http.Handler, bearerToken string, opts ...Option) http.Handler {
	options := options{}
	for _, opt := range opts {
		opt(&options)
	}
	router := chi.NewRouter()
	router.Get("/livez", probe(live))
	router.Get("/readyz", probe(ready))
	if metrics != nil {
		router.Handle("/metrics", metrics)
	}
	var handler http.Handler = router
	if bearerToken != "" {
		handler = requireBearerToken(handler, bearerToken)
	}
	if options.auditLogger != nil {
		handler = audit(handler, options.auditLogger, bearerToken != "")
	}
	return handler
}

func requireBearerToken(next http.Handler, token string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		const prefix = "Bearer "
		header := r.Header.Get("Authorization")
		if !strings.HasPrefix(header, prefix) || subtle.ConstantTimeCompare([]byte(strings.TrimPrefix(header, prefix)), []byte(token)) != 1 {
			w.Header().Set("WWW-Authenticate", `Bearer realm="goproxy-admin"`)
			http.Error(w, "unauthorized\n", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func audit(next http.Handler, logger *slog.Logger, authEnabled bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		status := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(status, r)
		auth := "disabled"
		if authEnabled {
			auth = "authorized"
			if status.status == http.StatusUnauthorized {
				auth = "unauthorized"
			}
		}
		logger.Info("admin request completed",
			"method", r.Method,
			"path", r.URL.Path,
			"status", status.status,
			"auth", auth,
			"remote_addr", remoteAddr(r.RemoteAddr),
			"duration_ms", float64(time.Since(started).Microseconds())/1000,
		)
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func remoteAddr(value string) string {
	host, _, err := net.SplitHostPort(value)
	if err != nil {
		return value
	}
	return host
}

func probe(check func() bool) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		if check == nil || !check() {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("not ready\n"))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	}
}
