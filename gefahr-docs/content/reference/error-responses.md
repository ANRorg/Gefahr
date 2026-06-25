---
title: Error responses
section: Reference
order: 230
summary: Stable JSON error codes returned by the public proxy and what operators should check first.
---

# Error responses

Gefahr returns stable JSON error bodies for proxy-generated failures.

```json
{"code":"rate_limited","message":"request rate limit exceeded"}
```

## Public error codes

| Code | Status | Meaning |
|---|---:|---|
| `ambiguous_request_path` | `400` | Path contains unsafe encoded separators or dot segments |
| `route_not_found` | `404` | No configured route matched host and path |
| `method_not_allowed` | `405` | Route policy rejected the method |
| `path_denied` | `403` | Route policy denied the path prefix |
| `required_header_missing` | `400` | Route policy required a missing header |
| `header_denied` | `403` | Route policy rejected a header |
| `query_too_large` | `414` | Query string exceeded route policy |
| `request_too_large` | `413` | Body exceeded configured limit |
| `rate_limited` | `429` | Per-route client budget exhausted |
| `proxy_overloaded` | `503` | Concurrency admission limit reached |
| `no_healthy_upstream` | `503` | No backend in the route pool is healthy |
| `bad_gateway` | `502` | Upstream transport failed |
| `upstream_timeout` | `504` | Upstream response header deadline elapsed |

## Headers

Rate-limited responses include `Retry-After`.

Method-policy denials include `Allow`.

All public responses include `X-Request-ID`.

## Debugging

Use the error code with request logs and metrics:

- `route_not_found`: inspect host and path.
- `rate_limited`: inspect trusted client identity.
- `no_healthy_upstream`: inspect backend health metrics.
- `proxy_overloaded`: inspect active backend requests and upstream latency.
- `bad_gateway`: inspect network, TLS, and upstream process logs.
