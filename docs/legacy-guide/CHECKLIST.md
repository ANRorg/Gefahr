# Checklist

Flat view of progress across the whole project. Tick boxes as phases complete.
Mirrors the acceptance criteria in each phase doc — this file is for at-a-glance
status, the phase docs are the source of truth for detail.

## Setup
- [ ] Go module initialized, project layout matches `01-ARCHITECTURE.md`
- [ ] Test fixture backend built (`test/fixtures/backend`)
- [ ] `10-DOCUMENTATION-STANDARDS.md` read before any code written

## Phase 1 — Bare TCP Proxy
- [ ] Listener accepts and dials a hardcoded backend
- [ ] Bidirectional byte copy works for tiny/medium/large payloads
- [ ] Clean shutdown on either side closing
- [ ] No fd/goroutine leak after 100 connections
- [ ] Doc comments + NOTES.md per `10-DOCUMENTATION-STANDARDS.md`

## Phase 2 — HTTP Parsing
- [ ] Manual request parser (request line, headers, Content-Length body)
- [ ] Manual chunked transfer-encoding support
- [ ] Manual response parser
- [ ] Header rewriting: Host, X-Forwarded-For, X-Forwarded-Proto
- [ ] Hop-by-hop headers stripped
- [ ] Malformed input handled without crashing
- [ ] Unit tests for parser edge cases passing
- [ ] Doc comments + NOTES.md per `10-DOCUMENTATION-STANDARDS.md`

## Phase 3 — Load Balancing
- [ ] Backend pool + pluggable Balancer interface
- [ ] Round-robin implemented and tested
- [ ] Least-connections implemented and tested
- [ ] Active health checks mark backends dead/alive correctly
- [ ] All-backends-dead case returns clean error, not a hang
- [ ] Race-tested (`-race`)
- [ ] Doc comments + NOTES.md per `10-DOCUMENTATION-STANDARDS.md`

## Phase 4 — Caching
- [ ] In-memory cache with TTL
- [ ] Cacheability rules: GET-only, 200-only, respects no-store/private,
      excludes Set-Cookie responses
- [ ] max-age parsed from Cache-Control when present
- [ ] Expired entries treated as miss and evicted
- [ ] Race-tested under concurrent access
- [ ] Doc comments + NOTES.md per `10-DOCUMENTATION-STANDARDS.md`

## Phase 5 — TLS
- [ ] Self-signed cert generated for local testing
- [ ] TLS listener working, MinVersion set to TLS 1.2+
- [ ] X-Forwarded-Proto reflects http vs https correctly
- [ ] Plain HTTP and HTTPS both functional (if running both)
- [ ] Doc comments + NOTES.md per `10-DOCUMENTATION-STANDARDS.md`

## Phase 6 — Hardening
- [ ] Read/write deadlines on client and backend connections
- [ ] Max header size enforced
- [ ] Max body size enforced (including during chunked reads)
- [ ] Graceful shutdown on SIGTERM/SIGINT with bounded drain
- [ ] Per-request logging (backend, status, latency, cache hit/miss)
- [ ] Doc comments + NOTES.md per `10-DOCUMENTATION-STANDARDS.md`

## Final pass
- [ ] Full integration test across all phases together
- [ ] Load test run, no goroutine/memory leak observed
- [ ] README written — a stranger could clone and run this
- [ ] Known limitations documented (Vary handling, no LRU eviction, no
      passthrough TLS, no HTTP/2, etc.)
- [ ] Every exported identifier passes a `go doc` legibility check
