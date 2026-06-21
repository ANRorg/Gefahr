# Testing

## Philosophy

Test each phase in isolation as you build it — don't wait until everything's
wired together to find out Phase 2's chunked-encoding handling was broken the
whole time. This file gives concrete setups, not just "write tests."

## Test fixtures you'll want early

**A trivial test backend**, reusable across phases. A 30-line Go program (or
even a Python one-liner) that:
- Listens on a configurable port.
- Echoes back which port it's on in the response body (so you can visually
  confirm load balancing is alternating between backends).
- Has a `/slow` route that sleeps N seconds (for testing timeouts and
  least-connections).
- Has a `/health` route returning 200 (for health checks).
- Optionally: a route that returns chunked transfer-encoding, and one that
  sets `Cache-Control` headers you control, for Phase 2 and Phase 4 testing.

Build this once, in `test/fixtures/backend/main.go`, and reuse it for every
phase below instead of rebuilding test infrastructure each time.

## Phase 1 tests

- Unit: none needed yet, this is mostly integration.
- Integration: start fixture backend, start proxy, send a TCP payload of
  increasing size (1B, 1KB, 1MB), assert byte-for-byte echo back.
- Manual: `curl` through the proxy; kill the backend mid-`curl` and confirm
  the proxy doesn't hang.

## Phase 2 tests

- Unit (`internal/httpmsg`): table-driven tests for `ParseRequest` covering:
  a simple GET with no body, a POST with `Content-Length`, a response with
  chunked encoding, a request with both `Content-Length` and
  `Transfer-Encoding` set (should error — smuggling protection), malformed
  request lines, headers with unusual but valid casing.
- Integration: `curl -v` through the proxy against the fixture backend for
  each of the above; inspect that `X-Forwarded-For`/`X-Forwarded-Proto` land
  correctly in the backend's logged headers.

## Phase 3 tests

- Unit (`internal/balancer`): for round-robin, assert that N calls to
  `Next()` with M backends visit each backend `N/M` times in order. For
  least-connections, assert it picks the backend with the lowest
  `ActiveConns` given a contrived set of counts.
- Integration: 3 fixture backends, hammer the proxy with sequential requests,
  assert the distribution matches the configured strategy. Kill one backend,
  confirm it drops out within one health-check interval; restart it, confirm
  it rejoins.
- Run with `go test -race` — concurrent access to `ActiveConns` and the
  round-robin index is exactly the kind of thing that looks fine until raced.

## Phase 4 tests

- Unit (`internal/cache`): `Set` then `Get` returns the same response; `Get`
  after TTL expiry returns a miss; `isCacheable` rejects POST, rejects
  `Set-Cookie` responses, rejects `Cache-Control: no-store`.
- Integration: request the fixture backend's `/slow` route twice through the
  proxy with caching on — first request should take ~N seconds, second
  should be near-instant (cache hit). Vary `Cache-Control` on the fixture
  backend and confirm caching behavior changes accordingly.
- Race test: hammer the same cacheable URL concurrently with `-race`.

## Phase 5 tests

- Integration: `curl -k https://localhost:PORT` succeeds; `curl` (no `-k`)
  fails on the untrusted self-signed cert (expected, not a bug).
- `openssl s_client -connect localhost:PORT -tls1` should fail to connect if
  `MinVersion` is set correctly to TLS 1.2+.
- Confirm `X-Forwarded-Proto` is `https` for TLS-originated requests.

## Phase 6 tests

- A client that opens a connection and writes nothing — confirm the proxy
  disconnects it after the read timeout (time the disconnect, don't just
  assume).
- A backend that accepts but never writes a response — confirm the proxy
  returns an error to the client within the configured timeout, not a hang.
- Send a request with `Content-Length: 99999999999` and a tiny actual body —
  confirm the proxy rejects it before attempting to allocate that much.
- Send `SIGTERM` to the proxy process mid-request (script this: start a slow
  request, send the signal after it starts, confirm the response still
  completes before the process exits).

## Load testing (final sanity pass, not per-phase)

Once everything is built, run a basic load test (`hey`, `wrk`, or even a
simple Go script spawning concurrent requests) against the full stack —
multiple backends, caching on, TLS on — and watch for:
- Goroutine leaks (compare goroutine count before/after a burst of traffic
  settles).
- Memory growth that doesn't level off (possible cache or connection leak).
- Error rate under sustained concurrent load.

This isn't about hitting a specific number — it's about confirming nothing
degrades catastrophically under load you didn't explicitly design for.
