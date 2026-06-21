# Phase 6 — Hardening

## Goal

Turn "works on my machine against well-behaved clients" into something that
survives slow, malicious, or simply unlucky traffic without falling over.
This is what separates a learning project from something you'd actually trust
in front of real traffic — don't skip it because the proxy "already works."

## Concepts to understand before coding

- **Slowloris-style attacks.** A client that opens a connection and sends
  headers one byte every few seconds can hold a connection (and a goroutine)
  open indefinitely if you have no read timeout. At scale this exhausts your
  server's resources with minimal attacker effort.
- **Unbounded reads are a memory DoS vector.** A client sending a request
  with a 10GB `Content-Length` (or an attacker just lying about it) can blow
  out your memory if you blindly `io.ReadFull` that many bytes. Cap it.
- **Backend dying mid-request needs a defined behavior**, not a hang. Without
  timeouts, a dead/hanging backend holds the client connection open forever.
- **Graceful shutdown.** When you stop the proxy (SIGTERM/SIGINT), in-flight
  requests should be allowed to finish (within a bound) rather than being cut
  off mid-response.

## What to build

1. **Timeouts, everywhere:**
   - `conn.SetReadDeadline` / `SetWriteDeadline` on both the client connection
     and the backend connection — covers slow reads/writes and dead backends.
   - An idle timeout for keep-alive connections (if you implemented
     keep-alive in Phase 2) — don't let an idle connection sit open forever.
2. **Size limits:**
   - Max header size (reject if the headers section exceeds, say, 16KB —
     this is roughly what nginx defaults to).
   - Max body size (configurable, reject with `413 Payload Too Large` if
     `Content-Length` exceeds it, and also enforce the limit while actually
     reading in case of chunked encoding lying about size).
3. **Graceful shutdown:**
   - Listen for `SIGTERM`/`SIGINT` via `signal.Notify`.
   - Stop accepting new connections immediately.
   - Track in-flight connections (a `sync.WaitGroup` works) and wait for them
     to finish, up to a bounded timeout (e.g. 30s), then force-close anything
     still running.
4. **Logging / minimal observability:**
   - Log each request: method, path, backend chosen, status code, latency,
     cache hit/miss. Plain stdout logging is enough — this isn't a metrics
     system, just enough to debug behavior.
   - Optionally: a simple counter for cache hit rate and per-backend request
     counts, printed periodically or exposed on a debug endpoint.

## Go APIs involved

- `net.Conn.SetReadDeadline` / `SetWriteDeadline` / `SetDeadline`
- `io.LimitReader` to cap how much you'll read regardless of what
  `Content-Length` claims
- `os/signal.Notify`, `context.WithTimeout` for shutdown coordination
- `sync.WaitGroup` to track in-flight connections during shutdown

## Common mistakes

- Setting a deadline once at connection accept time instead of resetting it
  per-read/write — a single deadline for the whole connection defeats
  keep-alive (a connection idle between requests would get killed even
  though it's behaving fine).
- Limiting `Content-Length`-declared size but not actually capping bytes read
  during chunked decoding — the declared size isn't authoritative, the
  actual bytes are what matter.
- Shutdown that just calls `os.Exit()` — that's not graceful, it's a synonym
  for "drop everything."
- Logging so much (or so little) that the logs are useless under real load —
  aim for one structured line per request, not a line per internal step.

## Acceptance criteria

- [ ] A client that connects and sends nothing (or one byte every 10s) gets
  disconnected by your read timeout, not held open indefinitely.
- [ ] A backend that accepts the connection but never responds causes the
  proxy to return an error to the client after a bounded wait, not a hang.
- [ ] A request claiming an absurd `Content-Length` is rejected before you
  attempt to buffer that much memory.
- [ ] `SIGTERM` while a request is in flight lets that request finish before
  the process exits; new connections are refused immediately.
- [ ] Logs show enough to answer "what backend served this request, was it a
  cache hit, how long did it take" after the fact.

## After this phase

Your proxy meets the manifesto's "definition of done." From here, the
remaining `08-TESTING.md` and `CHECKLIST.md` are your verification pass — go
confirm everything actually holds up before calling it finished.
