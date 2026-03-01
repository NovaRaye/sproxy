# sproxy

[中文](README.zh.md)

An IPv6 rotating HTTP proxy. Picks a random IPv6 address from a given CIDR block for each connection.

Supports two modes:

- **Proxy-only**: use an existing tunnel/interface, just run the proxy
- **Tunnel mode**: automatically set up a 6in4 tunnel (e.g. from [Hurricane Electric](https://tunnelbroker.net)), configure routing, and run the proxy

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/novaraye/sproxy/master/install.sh | sudo sh
```

## Usage

### Proxy-only mode

```bash
sudo sproxy --cidr <IPv6 CIDR>
```

### Tunnel mode

Parameters are available on your tunnel provider's details page (e.g. tunnelbroker.net).

```bash
sudo sproxy \
  --cidr        <routed IPv6 block>      \
  --remote      <tunnel server IPv4>     \
  --local       <your server IPv4>       \
  --client-ipv6 <tunnel client IPv6/64>
```

## Docker

```bash
docker run -d \
  --name sproxy \
  --restart unless-stopped \
  --privileged \
  --network=host \
  ghcr.io/novaraye/sproxy \
  --cidr        2001:db8:1::/48  \
  --remote      216.218.226.238  \
  --local       203.0.113.10     \
  --client-ipv6 2001:db8::/64    \
  --port        1080             \
  --username    user             \
  --password    pass
```

## Verify

Loop requests to check the exit IP — each response should be a different IPv6 address:

```bash
while true; do curl -x http://127.0.0.1:1080 -s https://api.ip.sb/ip -A Mozilla; done
```

## Options

| Flag             | Default   | Description                                                          |
| ---------------- | --------- | -------------------------------------------------------------------- |
| `--cidr`         | —         | IPv6 CIDR block to use for proxy IPs (required)                      |
| `--remote`       | —         | Tunnel server IPv4 (enables tunnel mode)                             |
| `--local`        | —         | Your server's public IPv4 (tunnel mode)                              |
| `--client-ipv6`  | —         | Tunnel client IPv6 with prefix, e.g. `2001:db8::2/64` (tunnel mode) |
| `--gateway-ipv6` | —         | Tunnel gateway IPv6 (derived from `--client-ipv6` if omitted)        |
| `--port`         | `1080`    | Proxy listen port                                                    |
| `--bind`         | `0.0.0.0` | Proxy bind address                                                   |
| `--username`     | —         | Auth username (optional)                                             |
| `--password`     | —         | Auth password (optional)                                             |
