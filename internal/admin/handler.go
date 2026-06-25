// Package admin exposes operational endpoints on a separate listener.
package admin

import (
	"context"
	"crypto/subtle"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

type principalContextKey struct{}

type options struct {
	auditLogger *slog.Logger
	credentials []Credential
}

// Option customizes the operational handler.
type Option func(*options)

// WithAuditLogger emits one structured event for every admin request.
func WithAuditLogger(logger *slog.Logger) Option {
	return func(options *options) {
		options.auditLogger = logger
	}
}

// Credential describes one bearer token and the admin scopes it grants.
type Credential struct {
	Name   string
	Token  string
	Scopes []string
}

// WithCredentials installs scoped bearer-token credentials.
func WithCredentials(credentials []Credential) Option {
	return func(options *options) {
		options.credentials = append([]Credential(nil), credentials...)
	}
}

// NewHandler builds liveness, readiness, and optional metrics routes.
func NewHandler(live, ready func() bool, metrics http.Handler, bearerToken string, opts ...Option) http.Handler {
	options := options{}
	for _, opt := range opts {
		opt(&options)
	}
	credentials := append([]Credential(nil), options.credentials...)
	if bearerToken != "" {
		credentials = append(credentials, Credential{Name: "legacy-admin", Token: bearerToken, Scopes: []string{"admin"}})
	}
	router := chi.NewRouter()
	router.Get("/livez", probe(live))
	router.Get("/readyz", probe(ready))
	if metrics != nil {
		router.Handle("/metrics", metrics)
	}
	var handler http.Handler = router
	if len(credentials) > 0 {
		handler = requireCredentials(handler, credentials)
	}
	if options.auditLogger != nil {
		handler = audit(handler, options.auditLogger, len(credentials) > 0)
	}
	return handler
}

func requireCredentials(next http.Handler, credentials []Credential) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name, scopes, ok := matchCredential(r, credentials)
		if !ok {
			w.Header().Set("WWW-Authenticate", `Bearer realm="goproxy-admin"`)
			http.Error(w, "unauthorized\n", http.StatusUnauthorized)
			return
		}
		if !scopeAllowed(scopes, requiredScope(r)) {
			*r = *r.WithContext(context.WithValue(r.Context(), principalContextKey{}, name))
			http.Error(w, "forbidden\n", http.StatusForbidden)
			return
		}
		withPrincipal := r.WithContext(context.WithValue(r.Context(), principalContextKey{}, name))
		*r = *withPrincipal
		next.ServeHTTP(w, withPrincipal)
	})
}

func matchCredential(r *http.Request, credentials []Credential) (string, []string, bool) {
	const prefix = "Bearer "
	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(header, prefix) {
		return "", nil, false
	}
	token := []byte(strings.TrimPrefix(header, prefix))
	for _, credential := range credentials {
		if credential.Token != "" && subtle.ConstantTimeCompare(token, []byte(credential.Token)) == 1 {
			return credential.Name, credential.Scopes, true
		}
	}
	return "", nil, false
}

func requiredScope(r *http.Request) string {
	if r.Method == http.MethodGet && (r.URL.Path == "/livez" || r.URL.Path == "/readyz") {
		return "health"
	}
	if r.Method == http.MethodGet && r.URL.Path == "/metrics" {
		return "metrics"
	}
	return "admin"
}

func scopeAllowed(scopes []string, required string) bool {
	for _, scope := range scopes {
		if scope == "admin" || scope == required || (scope == "read" && (required == "health" || required == "metrics")) {
			return true
		}
	}
	return false
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
			} else if status.status == http.StatusForbidden {
				auth = "forbidden"
			}
		}
		logger.Info("admin request completed",
			"method", r.Method,
			"path", r.URL.Path,
			"status", status.status,
			"auth", auth,
			"principal", adminPrincipal(r),
			"remote_addr", remoteAddr(r.RemoteAddr),
			"duration_ms", float64(time.Since(started).Microseconds())/1000,
		)
	})
}

func adminPrincipal(r *http.Request) string {
	if principal, ok := r.Context().Value(principalContextKey{}).(string); ok && principal != "" {
		return principal
	}
	return "none"
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
