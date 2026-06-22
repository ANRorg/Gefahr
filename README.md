# Gefahr

[![CI](https://github.com/anouar/goproxy/actions/workflows/ci.yml/badge.svg)](https://github.com/anouar/goproxy/actions/workflows/ci.yml)
[![License](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](LICENSE)

Gefahr is a configurable Go reverse proxy with host/path routing, round-robin
and least-connections balancing, active and passive health tracking, bounded
response caching, static TLS termination, structured logs, and Prometheus
metrics.

The data plane uses Go's maintained `httputil.ReverseProxy`; Gefahr owns the
policy around it. See [the architecture decision](docs/adr/0001-proxy-foundation.md)
and [architecture overview](docs/architecture.md).

## Quick start

Requirements: Go 1.25.11 or Docker with Compose.

Run two fixture backends in separate terminals:

```sh
go run ./test/fixtures/backend -address :9001 -name backend-1
go run ./test/fixtures/backend -address :9002 -name backend-2
```

Then run the proxy:

```sh
go run ./cmd/Gefahr -config configs/proxy.example.yaml
curl -H 'Host: localhost' http://127.0.0.1:8080/
curl http://127.0.0.1:9090/readyz
curl http://127.0.0.1:9090/metrics
```

Or run the complete demonstration:

```sh
docker compose up --build
curl http://localhost:8080/
```

The production image runs its health check against the private `/readyz`
endpoint by invoking the GoProxy binary's bounded `-healthcheck` mode; it does
not add a shell or HTTP utility to the distroless image.

## Configuration and operation

Configuration is strict YAML: unknown fields and unsafe values stop startup or
reject reload. Copy `configs/proxy.example.yaml`, adjust listeners, routes, and
backend URLs, then pass it with `-config`.

- `SIGHUP` validates and atomically reloads routes, pools, policies, logging,
  and TLS certificate contents. Existing requests finish on their old snapshot.
- Listener addresses, listener count, TLS mode, the admin address, public
  server timeouts, shutdown timeout, and maximum header size require a restart.
  The per-listener connection limit is also restart-only.
- `SIGINT` and `SIGTERM` stop acceptance and drain requests within
  `timeouts.shutdown`.
- The admin listener should remain private. It exposes `/livez`, `/readyz`, and
  `/metrics`.

See the [configuration reference](docs/configuration.md) and
[operations runbook](docs/operations.md).

## Development

```sh
make test
make test-race
make check
make test-integration # requires permission to open local TCP listeners
make acceptance       # static, race, unit, and real-socket integration checks
docker compose up --build -d
make load-check      # exercises the running demonstration stack
```

Release builds can inject identity with `make build VERSION=v1.0.0` or with
`GOPROXY_VERSION` and `GOPROXY_COMMIT` for Compose. Run `goproxy -version` to
inspect the embedded values.

Every repository commit is intentionally small and independently testable.
See the [release acceptance procedure](docs/release-acceptance.md) for the
complete final gate and expected evidence.

## Security model and limitations

- Client forwarding headers are discarded and rebuilt from the trusted
  connection metadata.
- Request headers, bodies, network operations, cache memory, and shutdown are
  bounded.
- Shared caching bypasses authenticated, cookie-bearing, personalized,
  non-200, `private`, `no-store`, `no-cache`, and `Vary` responses.
- Static PEM certificates are loaded from deployment storage and must never be
  committed.

Version 1 does not include HTTP/3, ACME, dynamic service discovery, distributed
caching, cache revalidation, `Vary` variants, a mutation API, or per-route
authentication. The response write timeout limits very long-lived streams;
WebSocket-specific behavior is not an acceptance target. See
[security and limitations](docs/security.md).

## License

GoProxy is licensed under the [Apache License 2.0](LICENSE).
