# go-alived

A lightweight VRRP (Virtual Router Redundancy Protocol) implementation in Go, designed as a simple alternative to keepalived.

## Features

- **VRRP Protocol**: RFC 3768/5798 compliant implementation
- **High Availability**: Automatic failover with priority-based master election
- **Health Checking**: TCP, HTTP/HTTPS, ICMP ping, and script-based checks
- **Easy Deployment**: Built-in install command with systemd/init.d/Docker Compose support
- **Hot Reload**: Configuration reload via SIGHUP without service restart
- **Zero Dependencies**: Single static binary, no runtime dependencies

## Installation

### Download Binary

Download the latest release from [GitHub Releases](https://github.com/loveuer/go-alived/releases):

```bash
# Linux amd64
curl -LO https://github.com/loveuer/go-alived/releases/latest/download/go-alived-linux-amd64
chmod +x go-alived-linux-amd64
sudo mv go-alived-linux-amd64 /usr/local/bin/go-alived

# Linux arm64
curl -LO https://github.com/loveuer/go-alived/releases/latest/download/go-alived-linux-arm64
chmod +x go-alived-linux-arm64
sudo mv go-alived-linux-arm64 /usr/local/bin/go-alived
```

### Build from Source

```bash
git clone https://github.com/loveuer/go-alived.git
cd go-alived
go build -o go-alived .
```

### Quick Install (Recommended)

```bash
# Install as systemd service (default)
sudo ./go-alived install

# Install as init.d service (for OpenWrt/older systems)
sudo ./go-alived install --method service

# Generate Docker Compose deployment files
./go-alived install --method docker
```

## Quick Start

### 1. Test Environment

```bash
# Check if your environment supports VRRP
sudo go-alived test

# Test with specific interface
sudo go-alived test -i eth0
```

### 2. Configure

Edit `/etc/go-alived/config.yaml`:

```yaml
global:
  router_id: "node1"

vrrp_instances:
  - name: "VI_1"
    interface: "eth0"          # Network interface
    state: "BACKUP"            # Initial state
    virtual_router_id: 51      # VRID (1-255, must match on all nodes)
    priority: 100              # Higher = more likely to be master
    advert_interval: 1         # Advertisement interval in seconds
    auth_type: "PASS"          # Authentication type
    auth_pass: "secret"        # Password (max 8 chars)
    virtual_ips:
      - "192.168.1.100/24"     # Virtual IP address(es)
```

### 3. Start Service

```bash
# Systemd
sudo systemctl daemon-reload
sudo systemctl enable go-alived
sudo systemctl start go-alived

# Init.d
sudo /etc/init.d/go-alived start

# Docker Compose
docker compose up -d
```

### 4. Verify

```bash
# Check service status
sudo systemctl status go-alived

# Check VIP
ip addr show eth0 | grep 192.168.1.100

# View logs
sudo journalctl -u go-alived -f
```

## Configuration

### Two-Node HA Setup Example

**Node 1 (Primary)**:
```yaml
global:
  router_id: "node1"

vrrp_instances:
  - name: "VI_1"
    interface: "eth0"
    state: "MASTER"
    virtual_router_id: 51
    priority: 100              # Higher priority
    advert_interval: 1
    auth_type: "PASS"
    auth_pass: "secret"
    virtual_ips:
      - "192.168.1.100/24"
```

**Node 2 (Backup)**:
```yaml
global:
  router_id: "node2"

vrrp_instances:
  - name: "VI_1"
    interface: "eth0"
    state: "BACKUP"
    virtual_router_id: 51
    priority: 90               # Lower priority
    advert_interval: 1
    auth_type: "PASS"
    auth_pass: "secret"        # Must match
    virtual_ips:
      - "192.168.1.100/24"     # Must match
```

### Health Checking

```yaml
vrrp_instances:
  - name: "VI_1"
    # ... other settings ...
    track_scripts:
      - "check_nginx"          # Reference to health checker

health_checkers:
  - name: "check_nginx"
    type: "tcp"
    interval: 3s
    timeout: 2s
    rise: 3                    # Successes to mark healthy
    fall: 2                    # Failures to mark unhealthy
    config:
      host: "127.0.0.1"
      port: 80
```

**Supported Health Check Types**:

| Type | Description | Config |
|------|-------------|--------|
| `tcp` | TCP port check | `host`, `port` |
| `http` | HTTP endpoint check | `url`, `method`, `expected_status` |
| `ping` | ICMP ping check | `host`, `count` |
| `script` | Custom script | `script`, `args` |

## Commands

```
go-alived [command]

Available Commands:
  run         Run the VRRP service
  test        Test environment for VRRP support
  install     Install go-alived as a system service (alias: i)
  help        Help about any command

Flags:
  -h, --help      help for go-alived
  -v, --version   version for go-alived
```

### run

```bash
go-alived run [flags]

Flags:
  -c, --config string   Path to config file (default "/etc/go-alived/config.yaml")
  -d, --debug           Enable debug mode
```

### test

```bash
go-alived test [flags]

Flags:
  -i, --interface string   Network interface to test
  -v, --vip string         Test VIP address (e.g., 192.168.1.100/24)
```

### install

```bash
go-alived install [flags]

Flags:
  -m, --method string   Installation method: systemd, service, docker (default "systemd")

Aliases:
  install, i
```

## Signals

| Signal | Action |
|--------|--------|
| `SIGHUP` | Reload configuration |
| `SIGINT` / `SIGTERM` | Graceful shutdown |

```bash
# Reload configuration
sudo kill -HUP $(pgrep go-alived)
```

## Environment Compatibility

| Environment | Support | Notes |
|-------------|---------|-------|
| Physical servers | Full | |
| KVM/QEMU/Proxmox | Full | |
| VMware ESXi | Full | Enable promiscuous mode |
| VirtualBox | Full | Bridged network + promiscuous mode |
| Docker | Limited | Use `install --method docker`, requires `--privileged --net=host` |
| OpenWrt/iStoreOS | Full | Use `--method service` for install |
| AWS/Aliyun/Azure | None | Multicast disabled |

> **Note**: VRRP requires multicast support (224.0.0.18). Most public clouds disable multicast at the network layer. Use cloud-native HA solutions instead.

## Troubleshooting

### Common Issues

**1. "permission denied" or "operation not permitted"**
```bash
# VRRP requires root privileges
sudo go-alived run -c /etc/go-alived/config.yaml
```

**2. "authentication failed"**
- Ensure `auth_pass` matches on all nodes
- Password is limited to 8 characters

**3. Both nodes become MASTER (split-brain)**
- Check network connectivity between nodes
- Verify `virtual_router_id` matches
- Ensure multicast traffic is allowed

**4. VIP not pingable after failover**
- Gratuitous ARP may be blocked
- Check switch/router ARP cache timeout

### Debug Mode

```bash
sudo go-alived run -c /etc/go-alived/config.yaml -d
```

## License

MIT License

## Contributing

Contributions are welcome! Please feel free to submit issues and pull requests.
