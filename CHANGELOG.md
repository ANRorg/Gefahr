# Changelog

All notable changes to GoProxy are documented here. The project follows
Semantic Versioning.

## [Unreleased]

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
