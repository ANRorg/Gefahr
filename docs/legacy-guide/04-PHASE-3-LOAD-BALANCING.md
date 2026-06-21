# Phase 3 — Multiple Backends + Load Balancing

## Goal

Replace the hardcoded single backend with a pool, a pluggable selection
strategy, and health checks so dead backends get skipped automatically.

## Concepts to understand before coding

- **Why an interface for the strategy?** Because "how do I pick a backend" is
  a genuinely swappable policy — round-robin, least-connections, weighted,
  random, IP-hash (for session stickiness) all answer the same question
  differently. Hardcoding one means rewriting the proxy core to change it.
- **Round-robin:** cycle through backends in order, wrapping around. Simplest
  possible strategy, stateless except an index.
- **Least-connections:** pick the backend with the fewest currently-active
  proxied connections. Requires tracking active connection counts per
  backend (this is why `Backend.ActiveConns` exists in the architecture doc).
- **Health checks:** a backend can be "configured" but not actually reachable
  (crashed, restarting, network partition). Two common approaches:
  - *Active*: periodically (e.g. every 5s) make a lightweight request (often
    `GET /health` or even just a TCP dial) to each backend; mark dead/alive
    based on the result.
  - *Passive*: mark a backend dead when a real proxied request to it fails;
    retry it after a cooldown.
  A solid implementation does both — active checks catch problems before a
  real user hits them, passive checks catch things active checks missed.

## What to build

1. `internal/backend/backend.go`: `Backend` struct (addr, alive flag, active
   connection counter — all safely concurrent-accessible).
2. `internal/backend/pool.go`: holds `[]*Backend`, exposes `Alive() []*Backend`
   (filtered list) for the balancer to choose from.
3. `internal/balancer/balancer.go`: the `Balancer` interface
   (`Next(backends []*Backend) *Backend`).
4. `internal/balancer/roundrobin.go`: round-robin implementation.
5. `internal/balancer/leastconn.go`: least-connections implementation.
6. `internal/backend/healthcheck.go`: a goroutine that periodically checks
   each backend and updates its `Alive` flag.
7. Wire the balancer into `proxy.handleConn`: instead of a hardcoded address,
   call `balancer.Next(pool.Alive())`, increment that backend's active count
   before dialing, decrement it when the connection finishes (this is what
   makes least-connections meaningful).
8. Make the strategy configurable (e.g. a config field `"strategy": "round_robin"`).

## Go APIs involved

- `sync/atomic` (`atomic.Bool`, `atomic.Int32` in modern Go) for `Alive` and
  `ActiveConns` without a full mutex.
- `time.NewTicker` for the periodic health-check loop.
- `context.Context` to cleanly stop the health-check goroutine on shutdown
  (this also previews Phase 6's graceful shutdown).

## Common mistakes

- Forgetting to decrement `ActiveConns` on *every* exit path (including
  errors) — leads to a backend looking permanently "busy" and starving it of
  traffic under least-connections.
- Round-robin index not being safe for concurrent access (multiple goroutines
  calling `Next()` at once) — needs atomic increment or a mutex.
- Health check marking a backend dead on a single failed check (flaky
  network blips cause flapping) — consider requiring N consecutive failures
  before marking dead, and M consecutive successes before marking alive
  again.
- All backends dead simultaneously — decide explicitly what the proxy returns
  to the client (a 502/503, not a hang or crash).

## Acceptance criteria

- [ ] Run 2–3 trivial backends locally (e.g. simple HTTP servers that each
  print their own port in the response body) and confirm round-robin visibly
  alternates between them across repeated requests.
- [ ] Kill one backend mid-test; confirm the proxy stops routing to it within
  one health-check interval and continues serving from the rest.
- [ ] Bring the killed backend back; confirm it rejoins the pool.
- [ ] Switch strategy to least-connections and confirm (e.g. with one slow
  backend and one fast one) that the slow one gets fewer new requests while
  it's still handling old ones.
- [ ] Stop all backends; confirm the proxy returns a clean error response,
  not a hang.

## What NOT to do yet

- No caching yet — Phase 4.
- No TLS yet — Phase 5.
- Don't build session-affinity (sticky sessions) unless you finish everything
  else first — it's a legitimate stretch goal, not core scope.
