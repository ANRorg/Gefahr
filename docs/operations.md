# Operations runbook

## Probes

- `/livez` reports process lifecycle.
- `/readyz` succeeds only when every configured pool has a healthy backend.
- `/metrics` uses Prometheus text exposition with request, duration, cache,
  retry, backend health, and active-request series.

The container image checks `/readyz` with `goproxy -healthcheck URL`. This mode
uses direct HTTP connections, rejects redirects and non-200 responses, and
exits within five seconds.

Bind the admin listener to loopback or a private management network. GoProxy
does not authenticate admin endpoints.

## Reload

Validate edits in a separate process where practical, then atomically replace
the configuration file and send `SIGHUP`. A rejected reload is logged and the
previous snapshot remains active. Certificate files are parsed before any new
certificate is published.

## Shutdown

Send `SIGTERM` or `SIGINT`. New connections stop, health workers are canceled,
and in-flight requests drain up to `timeouts.shutdown`. Container orchestrators
should provide a termination grace period longer than that value.

## Diagnosis

Each JSON request log includes request ID, route, backend, status, latency,
attempt count, and cache result. Use the returned `X-Request-ID` to correlate a
client failure. A `503` means no healthy backend was eligible or the configured
request-admission limit was reached; inspect the JSON error `code` to distinguish
those cases. `502` indicates a transport failure; `504` indicates the upstream
deadline elapsed.

Monitor backend health, retry rate, 5xx rate, cache outcomes, request latency,
process memory, and goroutine count. Repeated backend flapping usually means
probe thresholds are too aggressive or the health endpoint is not representative.
