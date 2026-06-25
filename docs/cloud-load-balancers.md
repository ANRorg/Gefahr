# Cloud load balancer notes

These notes are deployment starting points, not a substitute for validating the
exact provider SKU, region, ingress controller, TLS policy, and health-check
behavior used in production. Provider documentation referenced here was
reviewed on 2026-06-25:

- [AWS Application Load Balancer forwarded headers](https://docs.aws.amazon.com/elasticloadbalancing/latest/application/x-forwarded-headers.html)
- [Google Cloud external Application Load Balancer overview](https://docs.cloud.google.com/load-balancing/docs/https)
- [Google Cloud custom headers](https://docs.cloud.google.com/load-balancing/docs/https/custom-headers-global)
- [Azure Application Gateway request modifications](https://learn.microsoft.com/en-us/azure/application-gateway/how-application-gateway-works)

## AWS Application Load Balancer

Use an ALB target group that points at Gefahr's public listener. Keep the admin
listener off the target group and reachable only from private monitoring or
operator networks.

Provider behavior to account for:

- ALB appends or creates `X-Forwarded-For` by default.
- ALB can add client ports to `X-Forwarded-For` when client-port preservation
  is enabled.
- ALB health checks do not need access to the admin listener; point them at a
  public application route that proves the proxy and an upstream are both
  healthy.

Gefahr settings:

```yaml
client_ip:
  trusted_proxies:
    - <alb-subnet-cidr>
  headers:
    - X-Forwarded-For
```

Prefer security groups for network enforcement, and keep the configured CIDRs
as narrow as the ALB source ranges allow. Gefahr accepts valid `IP:port` and
`[IPv6]:port` entries in `X-Forwarded-For` for client identity extraction.

## Google Cloud Application Load Balancer

Use an external or internal Application Load Balancer for HTTP(S) traffic, and
send backend traffic to Gefahr's public listener. Keep admin traffic private.

Provider behavior to account for:

- Google Cloud Application Load Balancers are proxy-based Layer 7 load
  balancers.
- External Application Load Balancers append the client IP and load balancer IP
  to `X-Forwarded-For`; existing values can be present before those entries.
- Custom request headers can be configured on backend services, but those
  custom headers are not added to health check probes.

Gefahr settings:

```yaml
client_ip:
  trusted_proxies:
    - <gfe-or-proxy-only-source-cidr>
  headers:
    - X-Forwarded-For
```

For internal Application Load Balancers, the proxy-only subnet is the natural
trusted proxy boundary. For external Application Load Balancers, validate the
backend source ranges in your VPC/firewall model and avoid trusting broad VPC
CIDRs that include clients or unrelated workloads.

## Azure Application Gateway

Use Application Gateway for public HTTP(S) entry and route backend traffic to
Gefahr's public listener. Keep the admin listener out of backend pools.

Provider behavior to account for:

- Application Gateway adds forwarded request metadata, including
  `X-Forwarded-For`, `X-Forwarded-Proto`, and `X-Forwarded-Port`.
- Its `X-Forwarded-For` values can include client ports.
- Backend health probes should target a public route unless you deliberately
  expose a separate authenticated probe path through private networking.

Gefahr settings:

```yaml
client_ip:
  trusted_proxies:
    - <application-gateway-subnet-cidr>
  headers:
    - X-Forwarded-For
```

Use Network Security Groups to ensure only Application Gateway reaches the
public listener, and use private monitoring paths for admin endpoints.

## Cutover Checks

Before sending production traffic through any managed load balancer:

1. Confirm the source address that Gefahr sees belongs to the configured trusted
   proxy CIDR.
2. Send test requests with unique client IPs and verify backend
   `X-Forwarded-For` contains the expected sanitized client identity.
3. Verify `/metrics` shows the expected route, status, retry, and rate-limit
   series after traffic.
4. Confirm health checks fail when upstreams are down and recover when upstreams
   are healthy.
5. Confirm the admin listener is unreachable from the public internet.
