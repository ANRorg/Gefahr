---
title: Quickstart
section: Documentation
order: 20
summary: Run two fixture backends, start Gefahr, send traffic, and inspect health and metrics.
---

# Quickstart

This guide starts Gefahr locally with two fixture backends and the example
configuration.

## Requirements

- Go 1.25 or newer.
- A shell that can open local TCP ports.
- Docker with Compose, if you prefer the container demo.

## Start fixture backends

Open two terminals from the repository root:

```sh
go run ./test/fixtures/backend -address :9001 -name backend-1
```

```sh
go run ./test/fixtures/backend -address :9002 -name backend-2
```

Each backend returns a small response that identifies the backend instance.

## Start Gefahr

Open a third terminal:

```sh
go run ./cmd/goproxy -config configs/proxy.example.yaml
```

The example config listens on:

| Listener | Address | Purpose |
|---|---:|---|
| Public proxy | `:8080` | Receives application traffic |
| Admin | `127.0.0.1:9090` | Serves health, readiness, and metrics |

## Send a request

```sh
curl -H 'Host: localhost' http://127.0.0.1:8080/
```

Send the same request a few times. With the default example, traffic should
rotate across the fixture backends.

## Check readiness

```sh
curl http://127.0.0.1:9090/readyz
```

`/readyz` returns `200` only when every configured pool has at least one
healthy backend.

## Check metrics

```sh
curl http://127.0.0.1:9090/metrics
```

Look for:

- `goproxy_requests_total`
- `goproxy_backend_healthy`
- `goproxy_backend_active_requests`
- `goproxy_policy_denials_total`
- `goproxy_rate_limit_decisions_total`

## Run the container demo

```sh
docker compose up --build
curl http://localhost:8080/
```

The Compose demo builds the image, starts the proxy, starts fixture backends,
and exercises the private readiness health check from inside the container.

## Stop the demo

Press `Ctrl+C` in the running terminals. Gefahr handles `SIGINT` and `SIGTERM`
by stopping new accepts and draining in-flight requests up to
`timeouts.shutdown`.
