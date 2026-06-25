# Checklist

Archived reconciliation of the original from-scratch lab checklist. This file is
not the current product acceptance source for Gefahr. The implemented proxy
follows [ADR 0001](../adr/0001-proxy-foundation.md), which supersedes the lab's
manual parser and bare TCP constraints in favor of Go's maintained `net/http`
and `httputil.ReverseProxy` stack. See the current
[product-readiness status](../completion.md).

Items marked "superseded" were intentionally replaced by ADR 0001 rather than
left unfinished.

## Setup
- [x] Go module initialized, with the current layout documented in
      `docs/architecture.md`
- [x] Test fixture backend built (`test/fixtures/backend`)
- [x] Documentation standards retained as legacy guidance; current operational
      docs live under `docs/`

## Phase 1 — Bare TCP Proxy
- [x] Superseded: listeners are `net/http` servers, not a hardcoded TCP pipe
- [x] Superseded: message streaming is delegated to Go's HTTP stack
- [x] Graceful shutdown is implemented for managed public and admin servers
- [x] Load/leak smoke checking is documented in `docs/release-acceptance.md`
- [x] Exported identifiers have doc comments; durable notes live in ADRs and
      current docs

## Phase 2 — HTTP Parsing
- [x] Superseded: request parsing is delegated to Go's `net/http`
- [x] Superseded: chunked transfer handling is delegated to Go's `net/http`
- [x] Superseded: response parsing is delegated to Go's `net/http`
- [x] Forwarding headers are rebuilt by the proxy
- [x] Hop-by-hop handling is delegated to Go's reverse proxy path
- [x] Malformed and ambiguous request paths are rejected safely
- [x] Unit tests cover forwarding, malformed paths, limits, and errors
- [x] Exported identifiers have doc comments; durable notes live in ADRs and
      current docs

## Phase 3 — Load Balancing
- [x] Backend pool + pluggable Balancer interface
- [x] Round-robin implemented and tested
- [x] Least-connections implemented and tested
- [x] Active health checks mark backends dead/alive correctly
- [x] All-backends-dead case returns clean error, not a hang
- [x] Race-tested (`make acceptance`)
- [x] Exported identifiers have doc comments; durable notes live in ADRs and
      current docs

## Phase 4 — Caching
- [x] In-memory cache with TTL
- [x] Cacheability rules: GET-only, 200-only, respects no-store/private,
      excludes Set-Cookie responses
- [x] max-age parsed from Cache-Control when present
- [x] Expired entries treated as miss and evicted
- [x] Race-tested under concurrent access (`make acceptance`)
- [x] Exported identifiers have doc comments; durable notes live in ADRs and
      current docs

## Phase 5 — TLS
- [x] Self-signed certificate generation is covered by tests instead of
      committed PEM files
- [x] TLS listener working, MinVersion set to TLS 1.2+
- [x] X-Forwarded-Proto reflects http vs https correctly
- [x] Plain HTTP and HTTPS listeners are supported by configuration
- [x] Exported identifiers have doc comments; durable notes live in ADRs and
      current docs

## Phase 6 — Hardening
- [x] Read/write deadlines on client and backend connections
- [x] Max header size enforced
- [x] Max body size enforced
- [x] Graceful shutdown on SIGTERM/SIGINT with bounded drain
- [x] Per-request logging includes backend, status, latency, attempts, and
      cache result
- [x] Exported identifiers have doc comments; durable notes live in ADRs and
      current docs

## Final pass
- [x] Full integration test across all current product areas
- [x] Load test command and leak interpretation documented
- [x] README written so a stranger can clone and run this
- [x] Known limitations documented in `README.md` and `docs/security.md`
- [x] Exported identifiers have doc comments
