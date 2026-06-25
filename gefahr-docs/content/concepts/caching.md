---
title: Caching
section: Concepts
order: 100
summary: Configure bounded shared response caching and understand what Gefahr intentionally refuses to cache.
---

# Caching

Gefahr includes a conservative shared response cache. It is designed to reduce
safe repeated upstream reads, not to replace a CDN.

## Enable cache on a route

```yaml
routes:
  - name: api
    host: api.example.test
    path_prefix: /api
    pool: api
    strategy: round_robin
    cache:
      enabled: true
```

The cache is process-local and bounded globally:

```yaml
cache:
  max_entries: 1000
  max_bytes: 67108864
  default_ttl: 30s
```

## Cacheable responses

Gefahr only caches responses that are safe for a shared proxy cache.

It bypasses:

- Non-`GET` and non-`HEAD` requests.
- Authenticated requests.
- Cookie-bearing requests.
- Responses with `Set-Cookie`.
- Responses marked `private`, `no-store`, or `no-cache`.
- Responses with `Vary`.
- Partial responses.
- Responses with malformed freshness directives.

## Freshness

Gefahr uses upstream freshness when available. If the upstream does not provide
a usable freshness directive, `cache.default_ttl` is the fallback.

## Size bounds

`cache.max_bytes` limits total response body storage. The cache also has an
entry-count bound. Oversized or partially streamed responses are not published
to the cache.

## Operational notes

The cache is intentionally simple:

- It disappears on restart.
- It is not shared across replicas.
- It does not revalidate stale responses.
- It does not implement `Vary` variants.

Use a CDN or application cache when you need distributed cache behavior.
