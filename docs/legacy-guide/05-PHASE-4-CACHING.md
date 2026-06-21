# Phase 4 — Caching

## Goal

Cache safe, cacheable responses in memory so repeated identical requests skip
the backend entirely. This phase teaches the actually-hard part of caching:
deciding what's safe to cache and when to stop trusting it.

## Concepts to understand before coding

- **What's cacheable, minimally:** `GET` requests, `200 OK` responses, and
  nothing marked uncacheable. Don't cache `POST`/`PUT`/`DELETE` responses —
  these are not idempotent/safe by HTTP semantics.
- **Cache-Control matters.** At minimum respect:
  - `no-store` — never cache this response.
  - `private` — don't cache in a shared cache (a reverse proxy in front of
    multiple users is a shared cache).
  - `max-age=N` — overrides your default TTL for this response.
  Ignoring these isn't "simpler," it's incorrect — you'd be serving stale or
  sensitive data to the wrong people.
- **Set-Cookie is a red flag.** A response with `Set-Cookie` is usually
  per-user; caching it and serving it to other clients leaks one user's
  session to another. Treat as uncacheable by default.
- **Cache key.** Method + full URL (path + query string) is the simplest
  correct starting point. Vary headers (`Accept-Encoding`, `Accept-Language`)
  technically should factor in too (`Vary` header) — note this as a known
  simplification if you skip it, don't silently ignore it.
- **TTL vs invalidation are different problems.** TTL expiry is "stop
  trusting this after N seconds," invalidation is "I know this changed, drop
  it now." This project only needs TTL — active invalidation (e.g. on a
  backend PUT/DELETE) is a stretch goal, not core scope.

## What to build

1. `internal/cache/cache.go`:
   - `Cache` struct: `map[string]*entry` behind a `sync.RWMutex`.
   - `entry`: stores the serialized response (status, headers, body) and an
     `expiresAt time.Time`.
   - `Get(key string) (*httpmsg.Response, bool)` — returns `false` if missing
     *or* expired (treat expired as a miss, and evict it while you're there).
   - `Set(key string, resp *httpmsg.Response, ttl time.Duration)`.
2. A `isCacheable(req, resp) bool` function implementing the rules above.
3. A `ttlFor(resp) time.Duration` function: reads `max-age` if present,
   otherwise falls back to a configured default.
4. Wire into `proxy.handleConn`:
   - Before dialing a backend: if method is GET, check cache. Hit → write
     cached response directly to client, skip backend entirely.
   - After getting a backend response: if cacheable, store it.
5. Add a periodic sweep (or lazy eviction on `Get`) so expired entries don't
   accumulate forever in memory — lazy eviction on access is enough, a
   background sweeper is a nice-to-have, not required.

## Go APIs involved

- `sync.RWMutex` — `RLock`/`RUnlock` for reads (the common case), `Lock` for
  writes.
- `time.Time`, `time.Now().Add(ttl)`, `time.Now().After(expiresAt)`.
- `strconv.Atoi` again, this time for parsing `max-age=N` out of
  `Cache-Control`.

## Common mistakes

- Caching responses with `Set-Cookie` (covered above — don't).
- Caching error responses (4xx/5xx) by accident — only cache success
  responses unless you have a specific reason otherwise.
- Mutating a cached response in place when serving it (e.g. appending a
  header) — corrupts the cached copy for the next request. Always copy
  before mutating, or design responses to be immutable once cached.
- Unbounded cache growth — at minimum, TTL eviction; for real production use
  you'd also want a max-size eviction policy (LRU), worth a comment in your
  README even if you don't implement it.
- Race conditions: two concurrent requests for the same uncached URL both
  miss and both hit the backend — acceptable for this project (a stricter
  version would dedupe in-flight requests, that's a stretch goal, not core).

## Acceptance criteria

- [ ] Repeated identical `GET` requests show a measurable latency drop after
  the first request (cache hit skips the backend — verify by logging
  "HIT"/"MISS" and/or having your test backend sleep artificially).
- [ ] A response with `Cache-Control: no-store` is never served from cache.
- [ ] A response with `Set-Cookie` is never served from cache.
- [ ] Entries expire correctly — request again after TTL elapses and confirm
  it's a fresh backend hit, not a stale cache hit.
- [ ] `POST` requests are never cached or served from cache, full stop.
- [ ] Cache is safe under concurrent load (run with `-race` while hammering
  it with concurrent requests).

## What NOT to do yet

- No active invalidation, no `Vary` handling, no LRU eviction — note as known
  limitations in your README, don't silently skip without acknowledging.
