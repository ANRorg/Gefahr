# Security and known limitations

Gefahr treats HTTP parsing and framing as a security boundary and delegates it
to Go's standard library. It ignores client-provided forwarding chains unless
the direct peer matches a configured trusted proxy CIDR, and then only uses
supported headers from that trusted hop. Error responses do not expose transport
details.
Ambiguous paths containing encoded or double-encoded separators, backslashes,
or dot segments are rejected before routing so the proxy and backend cannot
normalize a route boundary differently.

Resource controls include maximum headers and bodies, public read/write/idle
deadlines, upstream dial and response-header deadlines, bounded retries, a
bounded cache, bounded per-route rate limiting, bounded client-identity state,
and bounded graceful shutdown.

Admin endpoints can require bearer-token authentication with
`admin.auth_token_env`, but they are still designed for private management
networks rather than direct internet exposure. Admin requests are audited with
path, status, auth result, remote address, and duration, without logging bearer
tokens.

TLS listeners require a certificate/key pair and TLS 1.2 or newer. Private keys
must be mounted read-only with least-privilege filesystem permissions. HTTPS
upstreams can use the host trust store, configured CA files, SNI overrides, and
client certificates for mTLS. Backend connections are direct and do not inherit
ambient `HTTP_PROXY` or `HTTPS_PROXY` settings.

The cache deliberately bypasses every response containing `Vary` because
variant keying is not implemented. It also lacks revalidation, stale serving,
and distributed coherence. Cache contents are process-local and disappear on
restart.

Gefahr has no role-based authorization layer, WAF, HTTP/3, ACME client, dynamic
service discovery, or configuration mutation API. Place the admin listener on a
trusted network and add external controls where those capabilities are required.
