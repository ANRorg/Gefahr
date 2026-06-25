# Product readiness status

Gefahr has a working reverse-proxy core, but it is not yet a finished product.
The current implementation follows the accepted
[proxy foundation ADR](adr/0001-proxy-foundation.md), not the original
from-scratch lab constraints archived under [`docs/legacy-guide`](legacy-guide/README.md).

## Shipped core

| Area | Status | Evidence |
|---|---|---|
| Routing | Shipped | Exact host matching, catch-all routes, longest boundary-safe path prefixes, and ambiguous path rejection are implemented and tested in `internal/routing` and `internal/proxy`. |
| Forwarding | Shipped | The data plane uses Go's maintained `httputil.ReverseProxy`; forwarding headers are discarded and rebuilt from trusted socket or trusted-proxy metadata before backend dispatch. |
| Load balancing | Shipped | Round-robin and least-connections balancers are implemented with no-healthy-backend handling and race-tested unit coverage. |
| Health | Shipped | Active probes and passive transport-failure ejection update backend eligibility; readiness requires every pool to have a healthy backend. |
| Caching | Shipped | The shared cache is TTL-based, LRU-bounded by entry count and bytes, and rejects unsafe methods, personalized requests, `Set-Cookie`, `private`, `no-store`, `no-cache`, malformed freshness directives, and `Vary` responses. |
| TLS | Shipped | Public listeners can terminate static PEM certificates with TLS 1.2 minimum; HTTPS upstreams support custom CA files, SNI override, and client certificates for mTLS. |
| Limits and timeouts | Shipped | Public servers enforce header, body, concurrency, connection, idle, read, write, upstream dial, upstream response, trusted-client route rate-limit, and shutdown bounds. |
| Reload | Shipped | `SIGHUP` validates and stages a complete replacement snapshot before atomic publication; rejected reloads retain the previous snapshot. |
| Observability | Shipped | JSON request logs, admin audit logs, `/livez`, `/readyz`, `/metrics`, request metrics, cache metrics, rate-limit decision metrics, retry metrics, and backend health/active gauges are implemented. |
| Admin auth | Shipped | Admin endpoints can require a bearer token loaded from an environment variable via `admin.auth_token_env`. |
| Kubernetes baseline | Shipped | A hardened manifest includes secret-backed admin auth, exec probes, read-only non-root pods, restricted admin networking, services, and a PodDisruptionBudget. |
| systemd baseline | Shipped | A hardened service unit and environment-file template cover VM and bare-metal deployments with non-root execution and host-level sandboxing. |
| Release integrity | Shipped | Release workflow publishes archives/images and generates GitHub provenance and SBOM attestations. |
| Operations | Partial | Docker Compose, Kubernetes and systemd baselines, distroless runtime image, executable health check mode, release acceptance, load-check instructions, and first-pass upgrade/incident runbooks exist. Disaster recovery drills and cloud-specific load-balancer examples are still thin. |

## Product gaps

| Gap | Why it matters |
|---|---|
| Deployment hardening guide | Kubernetes and systemd/VM deployment are covered; cloud load balancer examples and disaster recovery drills still need detail. |
| Traffic protection | Per-route rate limiting exists, but there is no WAF, bot classification, or adaptive abuse-control policy. |
| Access-control model | Admin auth is bearer-token only and audited; there is no role model or integration with external identity providers. |
| Release packaging | Tagged archives, images, SBOMs, and attestations exist; package-manager manifests and cosign signatures are not included. |
| Compatibility matrix | HTTP/1.1 and HTTP/2 behavior rely on Go's stack, but there is no documented compatibility test matrix across common clients, proxies, and cloud load balancers. |

## Superseded legacy requirements

The legacy guide intentionally taught a manual HTTP/1.1 proxy built on `net`,
`bufio`, `tls`, and `io`. ADR 0001 supersedes that constraint for the current
product: manual request/response parsers and the bare TCP proxy phase are not
version 1 requirements. The project delegates HTTP framing and parser security
boundaries to Go's standard `net/http` stack, while keeping routing, balancing,
health, cache policy, limits, reloads, telemetry, and lifecycle behavior in
Gefahr-owned code.

## Acceptance command

Run the current engineering gate from a clean worktree:

```sh
make acceptance
```

That command verifies formatting, `go vet`, all unit tests under the race
detector, and the real-socket integration suite under the race detector.
