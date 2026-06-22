# Security and known limitations

GoProxy treats HTTP parsing and framing as a security boundary and delegates it
to Go's standard library. It never combines client-provided forwarding chains
with trusted metadata. Error responses do not expose transport details.
Ambiguous paths containing encoded or double-encoded separators, backslashes,
or dot segments are rejected before routing so the proxy and backend cannot
normalize a route boundary differently.

Resource controls include maximum headers and bodies, public read/write/idle
deadlines, upstream dial and response-header deadlines, bounded retries, a
bounded cache, and bounded graceful shutdown.

TLS listeners require a certificate/key pair and TLS 1.2 or newer. Private keys
must be mounted read-only with least-privilege filesystem permissions. Upstream
HTTPS uses the host platform trust store; custom trust roots and mutual TLS are
not supported in version 1. Backend connections are direct and do not inherit
ambient `HTTP_PROXY` or `HTTPS_PROXY` settings.

The cache deliberately bypasses every response containing `Vary` because
variant keying is not implemented. It also lacks revalidation, stale serving,
and distributed coherence. Cache contents are process-local and disappear on
restart.

GoProxy has no admin authentication, authorization layer, rate limiter, WAF,
HTTP/3, ACME client, dynamic service discovery, or configuration mutation API.
Place the admin listener on a trusted network and add external controls where
those capabilities are required.
