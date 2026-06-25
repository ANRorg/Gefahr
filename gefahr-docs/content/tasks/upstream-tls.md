---
title: Configure HTTPS upstreams
section: Tasks
order: 140
summary: Use CA files, SNI overrides, and optional client certificates for upstream TLS.
---

# Configure HTTPS upstreams

Use HTTPS upstreams when traffic between Gefahr and backend services must be
encrypted or authenticated.

## Basic HTTPS backend

```yaml
pools:
  api:
    backends:
      - name: api-1
        url: https://api-1.internal:8443
```

Gefahr uses the host trust store by default.

## Custom CA

Use `tls.ca_file` when the upstream certificate is signed by a private CA:

```yaml
pools:
  api:
    tls:
      ca_file: /etc/goproxy/upstream-ca.pem
```

Mount the CA file read-only.

## SNI override

Use `tls.server_name` when the backend URL host does not match the certificate
name:

```yaml
pools:
  api:
    tls:
      server_name: api.internal
```

This value controls SNI and certificate hostname verification.

## Client certificates

Use upstream mTLS when the backend must authenticate Gefahr:

```yaml
pools:
  api:
    tls:
      ca_file: /etc/goproxy/upstream-ca.pem
      server_name: api.internal
      client_cert_file: /etc/goproxy/upstream-client.crt
      client_key_file: /etc/goproxy/upstream-client.key
```

`client_cert_file` and `client_key_file` must be configured together.

## Diagnostics only

`insecure_skip_verify` disables certificate verification. Use it only for
short isolated diagnostics, never as a production fix.

```yaml
tls:
  insecure_skip_verify: true
```

If this is needed in production, the real issue is usually missing CA material,
wrong SNI, or a certificate with the wrong names.
