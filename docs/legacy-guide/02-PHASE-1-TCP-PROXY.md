# Phase 1 — Bare TCP Proxy

## Goal

Prove you understand "proxy" at the socket level, with zero HTTP awareness. By
the end of this phase you have a dumb byte-pipe that works for *any* TCP
protocol, not just HTTP. This is the conceptual core every later phase builds on.

## Concepts to understand before coding

- A TCP connection is bidirectional and full-duplex: data can flow client→proxy
  and proxy→client at the same time, independently.
- A proxy, at minimum, is: accept a connection, open a *second* connection to
  somewhere else, and shuttle bytes between the two until either side closes.
- `io.Copy` blocks until EOF or error — to pipe both directions at once you need
  two goroutines (or two separate copy loops running concurrently).
- When one side closes its connection, the other side needs to know to stop too,
  or you leak goroutines and sockets.

## What to build

1. A `TCPListener` on a configurable port (hardcode `:8080` for now, config
   comes in `main.go` later).
2. On each `Accept()`, spawn a goroutine.
3. Inside that goroutine: `net.Dial("tcp", backendAddr)` to one fixed backend
   address (hardcode it — no pool yet, that's Phase 3).
4. Copy bytes both directions concurrently. Close both connections when either
   side is done.
5. Stand up a trivial backend to test against — even `python3 -m http.server`
   on another port works fine, or `nc -l` if you just want to watch raw bytes.

## Go APIs involved

- `net.Listen("tcp", addr)`, `Listener.Accept()`
- `net.Dial("tcp", addr)`
- `io.Copy(dst, src)`
- `sync.WaitGroup` or a channel, to know when both copy directions are finished
- `defer conn.Close()` — but be careful, see pitfalls below

## Common mistakes

- **Closing too early.** If you close the backend connection the moment the
  client→backend copy finishes, you may cut off a response still streaming
  back. Close only after *both* directions are done, or use
  `conn.CloseWrite()` (half-close) so the other direction can still finish.
- **Forgetting to close at all.** Every accepted connection that isn't
  eventually closed is a leaked file descriptor. Under load this will exhaust
  your OS's fd limit.
- **Blocking Accept() in the same goroutine as connection handling.** The
  accept loop itself must stay tight — `accept → go handle(conn) → accept`
  — or you can only serve one connection at a time.
- **Not testing with a slow/large transfer.** A proxy that works with `curl`
  on `localhost` for a tiny payload can still break on a multi-megabyte
  streamed response if you buffered something you shouldn't have. At this
  phase you shouldn't be buffering at all — `io.Copy` streams.

## Acceptance criteria (phase is done when...)

- [ ] You can `curl` (or any TCP client) through your proxy and reach the
  backend, for at least 3 different payload sizes (tiny, a few KB, a few MB).
- [ ] Closing the client mid-transfer doesn't hang or crash the proxy.
- [ ] Killing the backend mid-transfer results in the proxy closing the client
  connection cleanly (not hanging forever).
- [ ] You can explain, out loud or in writing, what each of the two `io.Copy`
  calls is doing and why you need both.
- [ ] No goroutine or fd leak after 100 sequential connections (check with
  `lsof` or simply watch goroutine count if you added a debug endpoint —
  optional but a good habit to start now).

## What NOT to do yet

- No HTTP parsing. If you find yourself inspecting bytes for `GET ` or
  `Host:`, you've jumped ahead to Phase 2.
- No multiple backends. One hardcoded address is correct for this phase.
- No TLS. Plain TCP only.
