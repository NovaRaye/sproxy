# sproxy

IPv6 随机出口 HTTP 代理。每次连接从指定的 CIDR 段中随机选取一个 IPv6 地址作为出口。

支持两种模式：

- **仅代理模式**：已有隧道/接口和路由，直接跑代理
- **隧道模式**：自动创建 6in4 隧道（如 [Hurricane Electric](https://tunnelbroker.net)）、配置路由并启动代理，退出时自动清理

## 安装

```bash
curl -L https://github.com/novaraye/sproxy/releases/latest/download/sproxy-linux-amd64 -o /usr/local/bin/sproxy
chmod +x /usr/local/bin/sproxy
```

## 使用

### 仅代理模式

已有隧道接口和路由配置时使用。

```bash
sudo sproxy --cidr <IPv6 地址段>
```

### 隧道模式

由 sproxy 自动创建 6in4 隧道并配置路由。
参数可在隧道提供商详情页找到（如 tunnelbroker.net）。

```bash
sudo sproxy \
  --cidr        <路由 IPv6 地址段>     \
  --remote      <隧道服务器 IPv4>      \
  --local       <本机公网 IPv4>        \
  --client-ipv6 <隧道客户端 IPv6/64>
```

### 参数说明

| 参数             | 默认值    | 说明                                                        |
| ---------------- | --------- | ----------------------------------------------------------- |
| `--cidr`         | —         | 代理出口 IPv6 地址段（必填）                                |
| `--remote`       | —         | 隧道服务器 IPv4（填写后启用隧道模式）                       |
| `--local`        | —         | 本机公网 IPv4（隧道模式必填）                               |
| `--client-ipv6`  | —         | 隧道客户端 IPv6 及前缀，如 `2001:db8::2/64`（隧道模式必填） |
| `--gateway-ipv6` | -         | 隧道网关 IPv6（默认从 `--client-ipv6` 推导）                |
| `--port`         | `1080`    | 代理监听端口                                                |
| `--bind`         | `0.0.0.0` | 代理绑定地址                                                |
| `--username`     | —         | 认证用户名（可选）                                          |
| `--password`     | —         | 认证密码（可选）                                            |
