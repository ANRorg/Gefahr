# systemd deployment

The unit in [`deploy/systemd/goproxy.service`](../deploy/systemd/goproxy.service)
is a hardened VM/bare-metal baseline for running Gefahr without Kubernetes. It
assumes:

- Binary: `/usr/local/bin/goproxy`
- Config: `/etc/goproxy/proxy.yaml`
- Secret env file: `/etc/goproxy/goproxy.env`
- Runtime user/group: `goproxy:goproxy`

Install a release archive and create the runtime identity:

```sh
sudo useradd --system --home-dir /nonexistent --shell /usr/sbin/nologin goproxy
sudo install -o root -g root -m 0755 goproxy /usr/local/bin/goproxy
sudo install -d -o root -g goproxy -m 0750 /etc/goproxy
sudo install -o root -g goproxy -m 0640 configs/proxy.example.yaml /etc/goproxy/proxy.yaml
sudo install -o root -g root -m 0644 deploy/systemd/goproxy.service /etc/systemd/system/goproxy.service
```

Create `/etc/goproxy/goproxy.env` with a long random token, and reference it in
the config with `admin.auth_token_env: GOPROXY_ADMIN_TOKEN`:

```sh
sudo install -o root -g goproxy -m 0640 deploy/systemd/goproxy.env.example /etc/goproxy/goproxy.env
sudoedit /etc/goproxy/goproxy.env
sudoedit /etc/goproxy/proxy.yaml
```

Start and inspect:

```sh
sudo systemctl daemon-reload
sudo systemctl enable --now goproxy
sudo systemctl status goproxy
sudo journalctl -u goproxy -f
```

Reload validates and stages a complete replacement config before publishing it:

```sh
sudo systemctl reload goproxy
```

Rollback is file based. Restore the previous binary or config, then either
reload for config-only changes or restart for restart-only settings:

```sh
sudo cp /etc/goproxy/proxy.yaml.previous /etc/goproxy/proxy.yaml
sudo systemctl reload goproxy
sudo install -o root -g root -m 0755 goproxy.previous /usr/local/bin/goproxy
sudo systemctl restart goproxy
```

The service uses `ProtectSystem=strict`, drops write access to the host
filesystem, restricts address families, denies new privileges, and limits the
process to `CAP_NET_BIND_SERVICE` so non-root Gefahr can bind ports below 1024.
If you only bind high ports, remove both capability lines from the unit.

Operational notes:

- Keep the admin listener on loopback or a private management interface.
- Store TLS keys under `/etc/goproxy` with group-readable permissions only when
  the `goproxy` group must read them.
- Set `TimeoutStopSec` higher than `timeouts.shutdown`.
- Use a host firewall or cloud security group to expose only intended public
  listener ports.
