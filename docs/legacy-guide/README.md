# Reverse Proxy From Scratch

> Archived guide: this directory is retained as the original learning lab. It is
> not the current product acceptance source for Gefahr. The implemented proxy
> follows [ADR 0001](../adr/0001-proxy-foundation.md), which intentionally uses
> Go's maintained `net/http` and `httputil.ReverseProxy` stack instead of manual
> HTTP parsing. See the current [product-readiness status](../completion.md).

A guided, phase-by-phase build of a production-usable HTTP reverse proxy in Go —
written entirely on top of standard-library primitives (`net`, `bufio`, `tls`,
`io`), with **no** `net/http/httputil.ReverseProxy` and no third-party proxy
library. The point is to actually understand what a reverse proxy does, not
just have one running.

This repo is documentation, not the proxy itself — it's the spec/guide you (or
a coding agent) build the real project from.

## What you end up with

- Manual HTTP/1.1 request & response parsing (including chunked transfer
  encoding) — built by hand, not delegated to `net/http`.
- A pluggable load-balancing layer (round-robin + least-connections) with
  active health checks.
- An in-memory response cache that respects `Cache-Control`, TTL, and
  `Set-Cookie` safety.
- TLS termination with a configurable minimum version.
- Timeouts, size limits, and graceful shutdown — the difference between a toy
  and something that survives a slow or malicious client.

## Start here

| File | What it's for |
|---|---|
| [`00-MANIFESTO.md`](./00-MANIFESTO.md) | North star — definition of done, scope, non-goals. Read first. |
| [`01-ARCHITECTURE.md`](./01-ARCHITECTURE.md) | System design, package layout, core types. Read before writing code. |
| [`02-PHASE-1-TCP-PROXY.md`](./02-PHASE-1-TCP-PROXY.md) | Bare TCP byte-pipe — no HTTP awareness yet. |
| [`03-PHASE-2-HTTP-PARSING.md`](./03-PHASE-2-HTTP-PARSING.md) | Manual HTTP/1.1 parsing, header rewriting. |
| [`04-PHASE-3-LOAD-BALANCING.md`](./04-PHASE-3-LOAD-BALANCING.md) | Backend pool, balancing strategies, health checks. |
| [`05-PHASE-4-CACHING.md`](./05-PHASE-4-CACHING.md) | In-memory response cache with TTL. |
| [`06-PHASE-5-TLS.md`](./06-PHASE-5-TLS.md) | TLS termination. |
| [`07-PHASE-6-HARDENING.md`](./07-PHASE-6-HARDENING.md) | Timeouts, size limits, graceful shutdown, logging. |
| [`08-TESTING.md`](./08-TESTING.md) | How to verify each phase, concrete test setups. |
| [`09-AGENT-RULES.md`](./09-AGENT-RULES.md) | **Read this before handing any of this to a coding agent.** Stops it from shortcutting via `httputil.ReverseProxy`. |
| [`10-DOCUMENTATION-STANDARDS.md`](./10-DOCUMENTATION-STANDARDS.md) | This is a lab, not a product — required doc-comment, NOTES.md, and README standards so the code itself teaches. |
| [`CHECKLIST.md`](./CHECKLIST.md) | Flat progress checklist across all phases. |
| [`GLOSSARY.md`](./GLOSSARY.md) | Terms (SNI, slowloris, chunked encoding, etc) scoped to this project. |

## How to use this

1. Read the manifesto, architecture, and documentation-standards docs.
2. Work through phases 1 → 6 **in order** — each one's acceptance criteria
   must pass before the next starts.
3. Use `08-TESTING.md` to verify as you go, not just at the end.
4. Document as you go per `10-DOCUMENTATION-STANDARDS.md` — doc comments and
   a `NOTES.md` per phase, not a write-up bolted on at the end.
5. Track progress in `CHECKLIST.md`.
6. If you're delegating implementation to a coding agent (Codex, etc), give it
   `09-AGENT-RULES.md` first — every time, every session.

## Scope

HTTP/1.1 only. No HTTP/2/3, no web UI, no distributed cache, no WebSocket
proxying — see `00-MANIFESTO.md` for the full non-goals list and why they're
excluded.

## License

Use however you like — this is a personal learning project's guide, not a
package meant for distribution.
