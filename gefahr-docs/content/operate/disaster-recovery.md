---
title: Disaster recovery
section: Operate
order: 180
summary: Plan and rehearse recovery for bad config, bad binary, secret loss, and provider or region failure.
---

# Disaster recovery

Gefahr does not store durable application data. Recovery is mostly about
restoring the right binary or image, config, secrets, TLS material, and load
balancer path.

## What must be backed up

| Asset | Why it matters |
|---|---|
| Config file | Defines routes, pools, limits, policy, TLS paths |
| Release artifact or image digest | Lets you restore a known-good proxy |
| Admin token source | Required for health, readiness, and metrics access |
| Public TLS certificates and keys | Required when Gefahr terminates TLS |
| Upstream CA and client certs | Required for HTTPS upstreams and mTLS |
| Deployment manifests or unit files | Recreates runtime shape |

## Suggested drills

Run these in staging before production:

1. Bad config rollback.
2. Bad binary or image rollback.
3. Admin token rotation.
4. Public TLS certificate replacement.
5. Upstream CA or mTLS material replacement.
6. Region or provider failover, if the service is multi-region.

## Evidence to record

For each drill, record:

- Start and end time.
- Operator.
- Environment.
- Artifact digest or binary checksum.
- Config checksum before and after.
- Commands run.
- Metrics checked.
- Smoke-test result.
- Whether RTO and RPO targets were met.

## RTO and RPO

Gefahr's own recovery point objective is usually config and secret freshness.
It does not own application data.

Recovery time depends on deployment path:

- Kubernetes rollback is usually fastest when previous ReplicaSets and secrets
  still exist.
- systemd rollback depends on local artifact and config availability.
- Region failover depends on DNS or load balancer control outside Gefahr.

Do not claim production readiness until recovery has been tested on the same
deployment path used for production.
