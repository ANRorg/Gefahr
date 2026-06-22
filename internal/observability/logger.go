// Package observability provides structured request logging.
package observability

import (
	"log/slog"
	"time"
)

// RequestLogger writes exactly one structured event per public request.
type RequestLogger struct{ Logger *slog.Logger }

// ObserveRequest emits the stable request log contract.
func (l RequestLogger) ObserveRequest(requestID, route, method, path, backend string, status, attempts int, cacheResult string, duration time.Duration) {
	logger := l.Logger
	if logger == nil {
		logger = slog.Default()
	}
	logger.Info("request completed", "request_id", requestID, "route", route, "method", method, "path", path, "backend", backend, "status", status, "attempts", attempts, "cache", cacheResult, "duration_ms", float64(duration.Microseconds())/1000)
}
