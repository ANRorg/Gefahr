---
title: Reloads and rollbacks
section: Operate
order: 160
summary: Safely change config, reload without dropping old requests, and return to a known-good version.
---

# Reloads and rollbacks

Gefahr supports atomic config reload with `SIGHUP`. Old requests continue on
the old snapshot. New accepted requests use the new snapshot after it is fully
validated and staged.

## Safe reload workflow

1. Generate or edit config outside the live path.
2. Run validation in a staging environment.
3. Replace the config atomically.
4. Send `SIGHUP` or run `systemctl reload goproxy`.
5. Inspect logs for reload success or failure.
6. Watch readiness, 5xx, retries, policy denials, and rate limits.

## What reload can change

Reload can change routes, pools, upstream TLS files, body limits, concurrency
limits, cache policy, and log level.

## What needs restart

Restart is required for public listener topology, listener TLS mode, admin
address, admin token environment variable, public server timeouts, shutdown
timeout, maximum header size, and the per-listener connection limit.

## Rollback config

Keep the previous config available with its checksum.

```sh
cp /etc/goproxy/proxy.previous.yaml /etc/goproxy/proxy.yaml
systemctl reload goproxy
```

Confirm:

- `/readyz` returns `200`.
- Public route smoke test succeeds.
- Error rate returns to baseline.
- The old config checksum is the active config.

## Rollback binary or image

For a binary, restore the previous executable and restart.

For Kubernetes, pin the previous image digest and wait for rollout status.

Record rollback evidence. A rollback procedure is not production-ready until
it has been run in staging.
