# Production transition checklist

Use this checklist before moving Gefahr from staging to production. It is
intentionally operational: the goal is to prove that the proxy, deployment
path, observability, and rollback process are ready together.

## Build And Release

- `make acceptance` passes on the release commit.
- `make coverage` passes the enforced coverage floor.
- Release artifacts, image digests, checksums, SBOMs, and attestations are
  generated and verified.
- The production deployment pins an exact image tag or binary checksum.
- The previous known-good version remains available for rollback.

## Configuration

- Config was reviewed and committed or otherwise approved.
- Route hosts and path prefixes match the intended traffic contract.
- `timeouts.*` values match expected client, load balancer, and upstream
  behavior.
- `limits.*` values are sized for production traffic and host capacity.
- `client_ip.trusted_proxies` contains only ingress/load-balancer source CIDRs.
- Per-route `policy` guardrails match the public API contract and ingress
  behavior.
- Per-route rate limits are configured only where the operational owner accepts
  the budget.
- Upstream TLS CA/SNI/client certificate settings are validated in staging.

## Deployment

- Kubernetes or systemd deployment path is selected and rehearsed.
- Admin listener is private and protected with `admin.auth_token_env` or scoped
  `admin.tokens[]`.
- Public TLS and upstream TLS secrets are mounted read-only.
- Health checks use either a public route with an upstream dependency or private
  `/readyz` with the admin bearer token.
- Graceful shutdown timeout is shorter than the orchestrator termination grace
  period.
- Load balancer idle timeout and proxy idle timeout are aligned.

## Observability

- Prometheus scrape path reaches `/metrics` through the private admin path.
- Logs are collected and searchable by request ID.
- Admin audit logs are searchable by `auth`, `principal`, `path`, and
  `remote_addr`.
- Dashboards include request rate, status codes, latency, retries, backend
  health, active backend requests, cache outcomes, policy denials, rate-limit
  decisions, memory, and goroutines.
- Alerts exist for readiness failure, elevated 5xx, rising retries, no healthy
  upstream, overload, unexpected policy denials, unexpected 429s, and
  unauthorized admin access.

## Rollout

- Staging passed the same release artifact and config.
- Canary or one-batch rollout criteria are defined.
- Rollout pauses on readiness failures, elevated 5xx, retry spikes, or
  unexpected rate limiting.
- Rollback command is known before starting rollout.
- Release owner records start time, version, config checksum, and decision
  points.

## Disaster Recovery

- Bad config rollback drill completed.
- Binary/image rollback drill completed.
- Admin token and TLS material recovery drill completed.
- Region/provider failure drill completed, when multi-region or multi-provider
  routing is part of production.
- RTO/RPO targets are accepted by the service owner.

## Production Acceptance

Record this evidence for the production cutover:

```text
release:
image digest or binary checksum:
config checksum:
environment:
operator:
deployment path:
load balancer or ingress:
trusted proxy CIDRs:
acceptance command output:
coverage output:
artifact attestation verification:
staging smoke evidence:
production smoke evidence:
dashboards checked:
alerts checked:
rollback version:
rollback command:
open risks:
go/no-go decision:
```
