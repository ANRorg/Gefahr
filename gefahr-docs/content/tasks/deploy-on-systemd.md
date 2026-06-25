---
title: Deploy on systemd
section: Tasks
order: 120
summary: Run Gefahr on a VM or bare-metal host with a locked-down service unit and explicit config paths.
---

# Deploy on systemd

Use systemd when Gefahr runs directly on a VM or bare-metal host.

## Files

Recommended layout:

```text
/usr/local/bin/goproxy
/etc/goproxy/proxy.yaml
/etc/goproxy/goproxy.env
/etc/goproxy/tls/
/var/lib/goproxy/
```

The environment file should contain secrets such as:

```sh
GOPROXY_ADMIN_TOKEN=replace-with-a-real-token
```

Use restrictive permissions:

```sh
sudo chown root:goproxy /etc/goproxy/goproxy.env
sudo chmod 0640 /etc/goproxy/goproxy.env
```

## Service expectations

The service should:

- Run as a non-root user when binding high ports.
- Use `NoNewPrivileges=true`.
- Restrict writable paths.
- Mount private keys read-only.
- Restart on failure.
- Use `ExecReload` to send `SIGHUP`.

## Reload config

```sh
sudo systemctl reload goproxy
```

A failed reload leaves the previous snapshot active. Always inspect logs after
reload:

```sh
journalctl -u goproxy -n 100 --no-pager
```

## Restart-only changes

Restart for listener addresses, listener TLS mode, admin address, public server
timeouts, maximum header size, and connection limit changes:

```sh
sudo systemctl restart goproxy
```

## Host firewall

Expose only the public listener to clients. Restrict the admin listener to
loopback or a private management network.
