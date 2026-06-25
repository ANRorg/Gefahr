# Changelog

All notable changes to Gefahr are documented here. The project follows
Semantic Versioning.

## [Unreleased]

### Added

- Automate checksummed cross-platform binary archives and multi-architecture
  GHCR images for tagged releases.
- Add optional bearer-token authentication for admin endpoints through
  `admin.auth_token_env`.
- Add structured admin audit logs for operational endpoint access.
- Add per-route request policy guardrails for allowed methods, denied path
  prefixes, required/denied headers, and query-string size.
- Add per-route, per-client fixed-window rate limiting.
- Add bounded Prometheus metrics for per-route request-policy denials.
- Add bounded Prometheus metrics for per-route rate-limit admission decisions.
- Add trusted-proxy-aware client identity extraction for rate limiting and
  rebuilt forwarding headers.
- Add real-socket HTTP/2 frontend/upstream compatibility tests and a documented
  compatibility matrix.
- Add upstream HTTPS trust controls, SNI override, and client certificates for
  backend mTLS.
- Add a hardened Kubernetes deployment baseline.
- Add a hardened systemd deployment baseline and production runbooks.
- Add cloud load balancer deployment notes for AWS ALB, Google Cloud
  Application Load Balancer, and Azure Application Gateway.
- Add production-transition checklist and disaster-recovery drill templates.
- Raise test coverage and enforce an 85% repository coverage floor in CI.

## [1.0.1] - 2026-06-22

### Fixed

- Parse Cache-Control directive lists without splitting commas inside quoted
  extension values, including escaped quotes, and reject malformed quotes.

## [1.0.0] - 2026-06-22

### Added

- Host and longest-path-prefix routing.
- Round-robin and least-connections balancing with active and passive health.
- Safe bounded retries and stable gateway errors.
- Bounded HTTP-aware LRU response caching.
- Static TLS termination with reloadable certificates and TLS 1.2 minimum.
- Strict YAML configuration with atomic SIGHUP reload.
- Structured JSON request logs and Prometheus metrics.
- Bounded request admission, connections, bodies, headers, timeouts, and
  graceful shutdown.
- Non-root distroless image, Compose demonstration, and executable container
  health check.
- Race-tested unit and real-socket integration suites plus a repeatable
  load/leak smoke check.
