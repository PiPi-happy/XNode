# XNode

[简体中文说明](README.zh-CN.md) · English

Fork of xboard-node for the official [Xboard](https://github.com/cedar2025/Xboard) panel. Supports `sing-box` / `xray-core` dual kernels.

> **Disclaimer**: This project is for educational and learning purposes only.

## Features

- Protocols: V2Ray family, Trojan, Shadowsocks, Hysteria2, TUIC, AnyTLS, and Mieru (through the Mieru author's MBox-enabled sing-box fork; no separate Mita service is required).
- Sync: WebSocket push + REST polling dual channel
- User controls: speed limit, device limit, alive-IP tracking, hot update
- Deploy modes: node mode, machine mode, standalone mode
- Multi-instance: single process binding multiple panels / nodes

## Install

### Docker

```bash
docker run -d --restart=always --network=host \
  -e apiHost=https://panel.com -e apiKey=TOKEN -e nodeID=1 \
  ghcr.io/pipi-happy/xnode:latest
```

The fork image is published by this repository's CI. Mount a local `config.yml` into
`/etc/xboard-node/config.yml` when using `kernel.fallbacks`.

For Mieru nodes, XNode routes to the Mieru author's MBox-enabled sing-box fork automatically; see [Mieru support](docs/mieru.md).

### Installer (Linux systemd)

```bash
# Node mode
curl -fsSL https://raw.githubusercontent.com/PiPi-happy/XNode/dev/install.sh | \
  sudo bash -s -- --mode node --panel https://panel.example.com --token TOKEN --node-id 1

# Machine mode
curl -fsSL https://raw.githubusercontent.com/PiPi-happy/XNode/dev/install.sh | \
  sudo bash -s -- --mode machine --panel https://panel.example.com --token TOKEN --machine-id 1
```

The installer downloads this fork's `dev` release by default. For reproducible
deployments, pass `--version <release-tag>` after publishing a tagged release.

## xbctl

Run `xbctl` after installation for help. Common commands:

```bash
xbctl list                          # list all instances
xbctl status                        # running status
xbctl bind add-node --panel URL --token TOKEN --node-id 1
xbctl bind add-machine --panel URL --token TOKEN --machine-id 1
xbctl bind remove-node --panel URL --node-id 1
xbctl service restart
```

## Configuration

Legacy single-panel config is fully compatible. Appending bindings auto-migrates to `instances` format. See `config.yml.example`.

Local fallback is configured only in `config.yml`, never from panel data. It applies to **Trojan + TCP + TLS**: ordinary TLS requests are forwarded to a local web service based on ALPN, while Trojan authentication traffic continues to the node. It can serve a normal website for ordinary web requests, but does not guarantee protection from every probing or blocking technique.

`node` and `machine` are panel management modes and are independent of the kernel type. Both support fallback. sing-box is the default kernel; Xray is also supported. Both kernels use this configuration:

```yaml
kernel:
  type: xray # or singbox
  fallbacks:
    - dest: "127.0.0.1:8080"
    - alpn: h2
      dest: "127.0.0.1:8082"
```

Run a local HTTP/1.1 service on 8080 and an h2c-capable service on 8082. XNode
forwards fallback traffic but does not install or manage the website service.
Use explicit `host:port` destinations, and do not repeat an ALPN. Bind the local
services to `127.0.0.1` so their backend ports are not exposed publicly.

Fallback rules survive compatible panel updates. Top-level `kernel.fallbacks`
is inherited by instances; nodes in the same machine instance share the rules.
Use multiple instances or a node-level override when nodes need different
destinations. Switching a node away from Trojan TCP+TLS is rejected while local
fallbacks are enabled. See [PRD](docs/PRD.md).

For Docker deployments, mount the config containing `kernel.fallbacks` at
`/etc/xboard-node/config.yml`. If the fallback targets use `127.0.0.1`, the XNode
container must also use the host network (`--network=host`); otherwise that
address refers to the container itself. The systemd installer sets the
health-check port to `65530` by default; Docker or manual configs listen only
when `health_port` is set. The unauthenticated endpoint binds only to
`127.0.0.1`; remote monitoring cannot connect directly. Use an SSH tunnel or a
restricted local proxy when remote monitoring is needed.

## Extensions

- Custom routes: [docs-custom-routes.md](docs-custom-routes.md)
- Custom outbounds: [docs-custom-outbounds.md](docs-custom-outbounds.md)
- DNS providers (ACME DNS-01): [docs-dns-providers.md](docs-dns-providers.md)

## License

MPL-2.0.
