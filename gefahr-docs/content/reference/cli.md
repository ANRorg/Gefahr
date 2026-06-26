---
title: CLI reference
section: Reference
order: 220
summary: Command-line modes for serving traffic, validating config, printing version metadata, and running bounded health checks.
---

# CLI reference

The main binary is `goproxy`.

## Run the proxy

```sh
goproxy -config /etc/goproxy/proxy.yaml
```

The process starts public listeners and the admin listener defined in config.

## Validate config

```sh
goproxy -config /etc/goproxy/proxy.yaml -check-config
```

Validation loads the YAML with the same strict parser used at startup, rejects
unknown fields and invalid values, and exits before opening listeners. It does
not resolve admin token environment variables, so CI and deployment pipelines
can validate config shape without access to production secrets.

## Print version

```sh
goproxy -version
```

Release builds include version and commit metadata.

## Health check mode

```sh
goproxy -healthcheck http://127.0.0.1:9090/readyz
```

Health check mode:

- Uses a bounded HTTP client.
- Rejects redirects.
- Requires a `200` response.
- Exits non-zero on failure.
- Reads `GOPROXY_ADMIN_TOKEN` and sends it as a bearer token when present.

This is useful for distroless containers where no shell or curl binary exists.

## Signals

| Signal | Behavior |
|---|---|
| `SIGHUP` | Validate and atomically reload reloadable config |
| `SIGINT` | Graceful shutdown |
| `SIGTERM` | Graceful shutdown |

Shutdown stops accepting new connections and drains in-flight requests up to
`timeouts.shutdown`.
