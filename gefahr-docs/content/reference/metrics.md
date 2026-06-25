---
title: Metrics reference
section: Reference
order: 210
summary: Prometheus metrics emitted by Gefahr, labels, cardinality rules, and common uses.
---

# Metrics reference

Metrics are exposed on the private admin listener at `/metrics`.

## Request metrics

```text
goproxy_requests_total{route,method,status}
```

Counts completed public requests. Methods are bounded to known HTTP methods or
`OTHER`. Route labels come from validated config, plus `unmatched` and
`retired`.

```text
goproxy_request_duration_seconds_count{route}
goproxy_request_duration_seconds_sum{route}
```

Tracks public request duration as a Prometheus summary-style count and sum.

## Cache metrics

```text
goproxy_cache_requests_total{route,result}
```

`result` values include `hit`, `miss`, and `bypass`.

## Request-protection metrics

```text
goproxy_policy_denials_total{route,reason}
```

Reasons:

- `method_not_allowed`
- `path_denied`
- `required_header_missing`
- `header_denied`
- `query_too_large`
- `other`

```text
goproxy_rate_limit_decisions_total{route,decision}
```

Decisions:

- `allowed`
- `limited`
- `other`

## Retry metrics

```text
goproxy_retries_total{route}
```

Counts additional upstream attempts after the first attempt.

## Backend metrics

```text
goproxy_backend_healthy{pool,backend}
goproxy_backend_active_requests{pool,backend}
```

Use these to detect backend outage, uneven load, and stuck requests.

## Runtime metrics

Gefahr exposes basic Go runtime metrics:

```text
go_goroutines
go_memstats_alloc_bytes
```

## Cardinality rules

Metric labels are bounded by configuration or enums. Gefahr does not use raw
path, client IP, query string, user agent, or arbitrary headers as metric
labels.
