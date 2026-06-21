# Glossary

Terms you'll hit while building this, scoped to exactly what this project
needs. Check here before going down a web-search rabbit hole.

**Reverse proxy** — a server that sits in front of one or more backend
servers and forwards client requests to them, returning the backend's
response to the client as if it came from the proxy itself. (Contrast with a
*forward proxy*, which sits in front of clients and forwards their requests
outward — e.g. a corporate web filter.)

**Load balancing** — distributing incoming requests across multiple backend
servers, by some strategy (round-robin, least-connections, etc), so no single
backend is overwhelmed.

**Round-robin** — a load-balancing strategy that cycles through backends in
fixed order, wrapping around.

**Least-connections** — a load-balancing strategy that routes each new
request to whichever backend currently has the fewest active connections.

**Health check** — a periodic probe (active) or failure-tracking (passive)
mechanism to detect whether a backend is actually reachable/working, so dead
backends can be skipped.

**TLS termination** — the proxy decrypts incoming HTTPS traffic and forwards
plain HTTP to the backend. Contrast with **TLS passthrough**, where the proxy
routes encrypted traffic without decrypting it, typically based on SNI.

**SNI (Server Name Indication)** — a field in the TLS ClientHello (sent
*before* encryption begins) indicating which hostname the client wants —
lets a proxy route to the right backend/cert even though the rest of the
handshake is encrypted.

**Content-Length** — an HTTP header declaring the exact byte length of the
message body, allowing the reader to know exactly when the body ends.

**Transfer-Encoding: chunked** — an alternative way of framing an HTTP body
when the total length isn't known upfront (e.g. streamed content); the body
is sent as a series of length-prefixed chunks, ending with a zero-length
chunk.

**Hop-by-hop header** — a header that applies only to a single connection
(e.g. proxy↔client) and should not be forwarded to the next hop (e.g.
proxy↔backend). Examples: `Connection`, `Keep-Alive`, `Transfer-Encoding`,
`Upgrade`.

**X-Forwarded-For** — a de facto standard header a proxy adds, recording the
original client's IP address, since the backend would otherwise only see the
proxy's IP.

**X-Forwarded-Proto** — a de facto standard header indicating whether the
original client request was `http` or `https`, since a TLS-terminating proxy
would otherwise hide this from the backend.

**Request smuggling** — a class of attack/bug arising from inconsistent
parsing of request boundaries (often via conflicting `Content-Length` and
`Transfer-Encoding` headers) between two systems (e.g. proxy and backend)
that disagree about where one request ends and the next begins.

**Slowloris** — a denial-of-service technique where an attacker opens many
connections and sends data extremely slowly, tying up server resources
(threads/goroutines/connections) that would otherwise time out quickly
against a well-behaved client.

**Cache-Control** — an HTTP header controlling caching behavior, e.g.
`no-store` (never cache), `private` (don't cache in a shared cache),
`max-age=N` (cache for N seconds).

**TTL (time-to-live)** — how long a cached entry is trusted before being
treated as stale and either revalidated or discarded.

**Graceful shutdown** — stopping a server such that in-flight work is allowed
to complete (within a bound) rather than being abruptly terminated.

**Keep-alive** — reusing a single TCP connection for multiple sequential
HTTP requests/responses, instead of opening a new connection per request.
