---
title: Create your first proxy
section: Getting started
order: 40
summary: Build a minimal route, backend pool, health check, and request policy.
---

# Create your first proxy

This page builds a small configuration that accepts traffic for
`api.example.test`, routes `/api` to two upstreams, checks backend health, and
protects the route with simple request guardrails.

## Minimal config

```yaml
listeners:
  - address: ":8080"

admin:
  address: "127.0.0.1:9090"
  auth_token_env: GOPROXY_ADMIN_TOKEN

routes:
  - name: api
    host: api.example.test
    path_prefix: /api
    pool: api
    strategy: round_robin
    policy:
      allowed_methods:
        - GET
        - HEAD
        - POST
      denied_path_prefixes:
        - /api/internal
      denied_headers:
        - X-Debug-Bypass
      max_query_bytes: 4096
    rate_limit:
      enabled: true
      requests: 300
      window: 1m
      max_keys: 10000

pools:
  api:
    backends:
      - name: api-1
        url: http://10.0.1.10:9000
      - name: api-2
        url: http://10.0.1.11:9000
    health:
      path: /health
      interval: 5s
      timeout: 1s
      unhealthy_threshold: 2
      healthy_threshold: 1
    retry:
      max_attempts: 2

timeouts:
  read_header: 10s
  read_body: 30s
  write: 2m
  idle: 60s
  shutdown: 30s
  dial: 5s
  response_header: 30s

limits:
  max_header_bytes: 16384
  max_body_bytes: 10485760
  max_concurrent_requests: 1024
  max_connections: 4096

cache:
  max_entries: 1000
  max_bytes: 67108864
  default_ttl: 30s

logging:
  level: info
```

## Test routing

```sh
GOPROXY_ADMIN_TOKEN=dev-token \
  goproxy -config proxy.yaml
```

```sh
curl -H 'Host: api.example.test' http://127.0.0.1:8080/api/users
```

Gefahr matches the host and path prefix, selects a healthy backend from the
`api` pool, rebuilds forwarding headers, and streams the response.

## Test route policy

The route allows only `GET`, `HEAD`, and `POST`.

```sh
curl -i -X DELETE -H 'Host: api.example.test' http://127.0.0.1:8080/api/users
```

Expected result:

```text
HTTP/1.1 405 Method Not Allowed
Allow: GET, HEAD, POST
```

Denied requests are counted in `goproxy_policy_denials_total` and do not reach
the upstream.

## Test readiness

```sh
curl -H "Authorization: Bearer $GOPROXY_ADMIN_TOKEN" \
  http://127.0.0.1:9090/readyz
```

Readiness fails when any configured pool has no healthy backend.
