# Reverse Proxy From Scratch — Manifesto

This folder is the single source of truth for building a real, production-usable
reverse proxy from first principles. Read this file first, every time you come back.

## Why this exists

Off-the-shelf reverse proxies (nginx, Caddy, Traefik, `httputil.ReverseProxy`) hide
the mechanics on purpose — that's their whole value proposition. This project does
the opposite: every layer is built by hand so the mechanics are visible. Nothing in
this build may use a library that does the *core* job for us (HTTP parsing, proxying,
load balancing). Standard library primitives (TCP sockets, `tls` package, `bufio`)
are fine. A pre-built reverse proxy is not.

## Definition of done

The project is complete when it satisfies all of these, not before:

1. Accepts plain TCP and terminates TLS on a configurable port.
2. Manually parses HTTP/1.1 requests and responses (no `net/http` server/handler,
   no `httputil.ReverseProxy`).
3. Forwards requests to one of several backend servers, selected by a pluggable
   load-balancing strategy (at least round-robin + one other).
4. Skips dead backends via health checks.
5. Caches safe, cacheable GET responses in memory with TTL + invalidation.
6. Has timeouts, size limits, and graceful shutdown — it does not fall over under
   a slow client or a dead backend.
7. Has a test suite that proves each phase works in isolation.
8. Has a README a stranger could use to run it against real backends.

Until all 8 are true, the project is "in progress," not "done." Don't let phase 6
get skipped because phases 1–5 felt like the finish line.

## Tech stack

**Go.** Decided once, not re-litigated. Reasons: real proxies (Caddy, Traefik) are
written in it, the standard library gives you raw TCP + TLS without a framework,
and goroutines make per-connection concurrency trivial — which matters because a
proxy's entire job is concurrency.

## How to use this folder

- `01-ARCHITECTURE.md` — the system design and package layout. Read before writing
  any code, even Phase 1.
- `02` through `07` — one file per build phase, in order. Each has: goal, concepts
  you must understand before coding, what to build, the Go APIs involved, common
  mistakes, and acceptance criteria (how you know the phase is actually done).
- `08-TESTING.md` — how to verify each phase, with concrete test setups.
- `09-AGENT-RULES.md` — read this **before** handing any of this to Codex or any
  other coding agent. It exists because agents left unsupervised will "helpfully"
  reach for `net/http/httputil.ReverseProxy` and silently defeat the entire point.
- `CHECKLIST.md` — flat checklist mirroring all phases. Tick boxes as you go. This
  is the file to glance at to answer "where am I."
- `GLOSSARY.md` — terms you'll hit and may not know yet (SNI, slowloris, chunked
  transfer encoding, etc). Look things up here before searching the web; it's
  scoped to exactly what this project needs.

## Operating principles for the whole build

- **No magic.** If you don't understand what a line of code does, stop and find
  out before moving on. The proxy working is not the goal; understanding it is.
- **Phases are sequential.** Don't start Phase 3 (load balancing) with a Phase 2
  (HTTP parsing) that doesn't actually work yet. Each phase's acceptance criteria
  must pass before the next phase starts.
- **Correctness over cleverness.** A reverse proxy that's simple and correct beats
  one that's "optimized" and subtly broken. Get it right, then make it fast.
- **Every phase ends with something you can run and poke at**, not just code that
  compiles. If a phase doesn't produce a runnable, testable artifact, the phase
  isn't finished.

## Non-goals (explicitly out of scope)

- HTTP/2 or HTTP/3 support. HTTP/1.1 only. (Worth doing as a v2 project, not this one.)
- A web UI / admin dashboard.
- Distributed caching (Redis, etc) — in-memory only.
- WebSocket proxying — mention as a stretch goal at the very end, not core scope.

Keeping these out is deliberate — scope creep is how "learn how a proxy works"
projects die unfinished.
