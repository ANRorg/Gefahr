# Configuration reference

The complete example is [`configs/proxy.example.yaml`](../configs/proxy.example.yaml).
Durations use Go syntax such as `250ms`, `30s`, and `2m`.
Configuration input is capped at 4 MiB and topology counts are bounded to
prevent pathological startup or reload memory use.

`SIGHUP` reloads routes, pools, upstream timeouts, body limits, cache policy,
logging, and certificate contents. Listener topology, TLS mode, the admin
address, public server timeouts, shutdown timeout, and maximum header size are
restart-only, as is the per-listener connection limit.

## Top-level fields

- `listeners`: one or more public addresses. Optional `tls.cert_file` and
  `tls.key_file` enable TLS 1.2+ for that listener.
- `admin.address`: private liveness, readiness, and metrics listener.
- `routes`: ordered route definitions with unique names.
- `pools`: named backend groups referenced by routes.
- `timeouts`: public read/write, upstream dial/response, idle, and shutdown
  bounds.
- `limits`: maximum request header/body bytes, concurrent admitted requests,
  and connections per public listener.
- `cache`: process-wide maximum entries, bytes, and fallback TTL.
- `logging.level`: `debug`, `info`, `warn`, or `error`.

## Routes

`host` is an exact case-insensitive host without semantic wildcard expansion.
An empty host is a catch-all. `path_prefix` must begin with `/`; `/api` matches
`/api` and `/api/...`, not `/apix`. The longest matching prefix wins.

`strategy` is `round_robin` or `least_connections`. `rewrite_host: true` sends
the selected backend host; the default preserves the original request host.
`cache.enabled` opts the route into conservative shared response caching.

## Pools

Every backend requires a unique `name` and absolute `http` or `https` URL.
Backend URLs cannot contain embedded credentials or fragments.
Health settings define the probe path, interval, timeout, and consecutive
failure/success thresholds. `retry.max_attempts` is `1` or `2`; retries apply
only to safe replayable methods and only before response commitment.

Unknown fields, duplicate routes/listeners, nonexistent pool references,
invalid URLs, non-positive limits, and invalid durations are rejected together
with field-oriented errors. Route, pool, and backend identifiers use at most
128 ASCII letters, digits, dots, underscores, and hyphens.
