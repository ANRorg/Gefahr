# ADR 0001: Use Go's reverse-proxy foundation

## Status

Accepted.

## Decision

The data plane uses `net/http` and `net/http/httputil.ReverseProxy`. The admin
plane uses `chi`. GoProxy owns routing, balancing, health, caching,
configuration, limits, telemetry, and lifecycle behavior.

The guide under `docs/legacy-guide` remains useful background, but its ban on
Go's HTTP server and reverse proxy is superseded. Hand-written HTTP parsing is
not appropriate for this production-minded version: parser disagreement and
message framing are security boundaries best delegated to the standard
library.

## Consequences

- Requests and responses stream instead of being buffered by default.
- Go's HTTP/1.1 and HTTP/2 protocol implementations receive upstream fixes.
- The project remains responsible for safe forwarding headers, backend
  selection, retry rules, cache policy, bounded resources, and operations.
- HTTP/3, ACME, distributed caching, service discovery, and a mutable admin API
  are outside version 1.

