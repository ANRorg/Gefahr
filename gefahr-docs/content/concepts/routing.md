---
title: Routing
section: Concepts
order: 70
summary: Configure host and path-prefix matches that behave predictably at route boundaries.
---

# Routing

Routes decide which backend pool receives a request. A route matches on host
and path prefix.

## Host matching

`host` is exact and case-insensitive:

```yaml
routes:
  - name: api
    host: api.example.test
    path_prefix: /api
    pool: api
    strategy: round_robin
```

`api.example.test` and `API.EXAMPLE.TEST.` are treated as the same host.
Gefahr does not expand wildcards.

Use an empty host only when you intentionally want a catch-all virtual host:

```yaml
host: ""
```

## Path-prefix matching

`path_prefix` must start with `/`.

`/api` matches:

- `/api`
- `/api/`
- `/api/users`

`/api` does not match:

- `/apix`
- `/v1/api`

The longest matching path prefix wins.

## Avoid route ambiguity

Gefahr rejects route prefixes with ambiguous separators or segments. Do not use
encoded slashes, backslashes, dot segments, or double-encoded separators in
route boundaries.

Good:

```yaml
path_prefix: /api/internal
```

Rejected:

```yaml
path_prefix: /api/../internal
```

## Route ordering

Ordering only matters when two routes have the same match specificity. Prefer
unique, explicit route boundaries instead of relying on order.

## Preserving or rewriting Host

By default, Gefahr preserves the original request host when forwarding upstream.

Set `rewrite_host: true` when the upstream expects its own backend host:

```yaml
routes:
  - name: api
    host: api.example.test
    path_prefix: /
    pool: api
    strategy: round_robin
    rewrite_host: true
```

Be explicit about this during production review. Host preservation affects
upstream virtual hosting, logs, auth middleware, and redirects.
