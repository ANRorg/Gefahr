# Phase 5 — TLS Termination

## Goal

Accept HTTPS from clients, decrypt at the proxy, talk plaintext HTTP to
backends (the standard "TLS termination" pattern most reverse proxies use).

## Concepts to understand before coding

- **TLS termination vs passthrough.** Termination: the proxy holds the
  cert/key, decrypts incoming traffic, and the backend never sees TLS at all
  — it just gets plain HTTP. Passthrough: the proxy doesn't decrypt, just
  routes the encrypted bytes based on the SNI (Server Name Indication) field
  visible in the unencrypted ClientHello. **Build termination first** — it's
  what almost every reverse proxy does by default, and it's what lets you
  also do header rewriting / caching / load balancing on the decrypted
  request. Passthrough is a stretch goal at the very end.
- **Why the backend connection is typically still plaintext.** If proxy and
  backend are on the same trusted network (e.g. same machine, same private
  VPC), re-encrypting is often skipped for performance. In zero-trust setups
  people do re-encrypt to the backend ("TLS everywhere") — worth a comment in
  your README, not required for this build.
- **What a cert/key pair actually is.** A self-signed cert is fine for local
  development — it just won't be trusted by default by real browsers/clients
  (you'll need `-k`/`--insecure` with curl, or add it to a trust store).
  `crypto/tls` in Go needs a cert file and a private key file.
- **`X-Forwarded-Proto` matters here.** Once the proxy is terminating TLS,
  the backend has no idea the original request was HTTPS unless you tell it
  via this header (you should already have built this in Phase 2 — now it
  actually has a reason to be correct).

## What to build

1. Generate a self-signed cert for local testing:
   ```
   openssl req -x509 -newkey rsa:2048 -keyout key.pem -out cert.pem \
     -days 365 -nodes -subj "/CN=localhost"
   ```
2. Swap your plain `net.Listen("tcp", addr)` for `tls.Listen("tcp", addr, cfg)`
   where `cfg := &tls.Config{Certificates: []tls.Certificate{cert}}` and
   `cert, _ := tls.LoadX509KeyPair("cert.pem", "key.pem")`.
3. Everything downstream of `Accept()` is unchanged — `tls.Listen` returns a
   `net.Listener` whose connections are already decrypted by the time you
   read from them. This is the payoff of building Phase 1–4 on the plain
   `net.Conn` interface instead of something TLS-specific.
4. Make TLS optional via config — support running either plain HTTP or HTTPS
   (or both, on different ports) so you can compare/test easily.
5. Set `X-Forwarded-Proto: https` when serving over the TLS listener (vs
   `http` on the plain listener).

## Go APIs involved

- `tls.LoadX509KeyPair(certFile, keyFile string)`
- `tls.Listen(network, addr string, config *tls.Config)`
- `tls.Config` — also where you'd set minimum TLS version
  (`MinVersion: tls.VersionTLS12` at least) if you want to be a responsible
  proxy and not allow ancient, broken TLS versions.

## Common mistakes

- Forgetting `MinVersion` and accidentally allowing TLS 1.0/1.1 (deprecated,
  insecure) — set it explicitly.
- Testing with `curl` against a self-signed cert without `-k` and getting
  confused by the certificate error — this is curl correctly distrusting an
  untrusted self-signed cert, not a bug in your proxy.
- Hardcoding the cert/key path instead of making it configurable — you'll
  want different certs for local dev vs anything resembling real deployment.
- Mixing up which hop is encrypted — double check with a packet capture
  (`tcpdump`/Wireshark on loopback) that proxy→backend traffic is genuinely
  plaintext if that's your intended design, not still encrypted by accident
  or, worse, intended-plaintext-but-actually-leaking something.

## Acceptance criteria

- [ ] `curl -k https://localhost:PORT/...` through your proxy reaches the
  (plaintext) backend successfully.
- [ ] `X-Forwarded-Proto` is `https` on requests that came in over TLS, `http`
  on requests that came in over plain TCP (if you're running both).
- [ ] Old TLS versions are rejected (`openssl s_client -tls1` against your
  proxy should fail if you set `MinVersion` correctly).
- [ ] You can explain, in your own words, the difference between termination
  and passthrough, and why this project uses termination.

## What NOT to do yet (stretch goals, do these last if at all)

- TLS passthrough via SNI peeking.
- Re-encrypting proxy→backend traffic.
- Automatic cert provisioning (ACME/Let's Encrypt) — interesting but a whole
  separate project's worth of complexity.
