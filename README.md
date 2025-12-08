# go-alived

A lightweight, dependency-free VRRP (Virtual Router Redundancy Protocol) implementation in Go, designed as a simple alternative to keepalived.

## Features

✅ **Phase 1: Core VRRP Functionality (Completed)**
- VRRP protocol implementation (RFC 3768/5798)
- Virtual IP management (add/remove VIPs)
- State machine (INIT/BACKUP/MASTER/FAULT)
- Priority-based master election
- Gratuitous ARP for network updates
- Raw socket VRRP packet send/receive
- Timer management (advertisement & master-down timers)
- VRRP instance manager with multi-instance support
- Configuration hot-reload (SIGHUP)

✅ **Phase 2: Health Checking (Completed)**
- Health checker interface with rise/fall logic
- TCP health checks
- HTTP/HTTPS health checks
- ICMP ping checks
- Script-based checks (custom commands)
- Periodic health check scheduling
- Health check integration with VRRP priority
- Track scripts: automatic priority adjustment on health changes

🚧 **Phase 3: Enhanced Features (Planned)**
- State transition scripts (notify_master/backup/fault)
- Email/Webhook notifications
- Sync groups
- Virtual MAC support
- Metrics export

## Installation

### Build from source

```bash
git clone https://github.com/loveuer/go-alived.git
cd go-alived
go build -o go-alived .
```

## Quick Start

### 1. Test Your Environment

Before deployment, test if your environment supports VRRP:

```bash
# Basic test (auto-detect network interface)
sudo ./go-alived test

# Test specific interface
sudo ./go-alived test -i eth0

# Full test with VIP
sudo ./go-alived test -i eth0 -v 192.168.1.100/24
```

### 2. Run the Service

```bash
# Run with minimal config
sudo ./go-alived run -c config.mini.yaml -d

# Run with full config
sudo ./go-alived -c config.yaml

# Install as systemd service
sudo ./deployment/install.sh
sudo systemctl start go-alived
```

## Usage

### Commands

```
go-alived              # Run VRRP service (default)
go-alived run          # Run VRRP service
go-alived test         # Test environment for VRRP support
go-alived --help       # Show help
go-alived --version    # Show version
```

### Global Flags

```
-c, --config string    Path to configuration file (default "/etc/go-alived/config.yaml")
-d, --debug            Enable debug mode
-h, --help             Show help
-v, --version          Show version
```

### Test Command Flags

```
-i, --interface string    Network interface to test (auto-detect if not specified)
-v, --vip string          Test VIP address (e.g., 192.168.1.100/24)
```

See [USAGE.md](USAGE.md) for detailed usage documentation.

## Configuration

### Minimal Configuration

```yaml
# config.mini.yaml - VRRP only
global:
  router_id: "node1"

vrrp_instances:
  - name: "VI_1"
    interface: "eth0"
    state: "BACKUP"
    virtual_router_id: 51
    priority: 100
    advert_interval: 1
    auth_type: "PASS"
    auth_pass: "secret123"
    virtual_ips:
      - "192.168.1.100/24"
```

### Full Configuration Example

See `config.example.yaml` for complete configuration with health checking.

### Signals

- `SIGHUP`: Reload configuration
- `SIGINT/SIGTERM`: Graceful shutdown

## Architecture

```
go-alived/
├── main.go                 # Application entry point
├── internal/
│   ├── cmd/               # Cobra commands
│   │   ├── root.go        # Root command
│   │   ├── run.go         # Run service command
│   │   └── test.go        # Environment test command
│   ├── vrrp/              # VRRP implementation
│   │   ├── packet.go      # VRRP packet structure & marshaling
│   │   ├── socket.go      # Raw socket operations
│   │   ├── state.go       # State machine & timers
│   │   ├── arp.go         # Gratuitous ARP
│   │   ├── instance.go    # VRRP instance logic
│   │   └── manager.go     # Instance manager
│   └── health/            # Health check system
│       ├── checker.go     # Checker interface & state
│       ├── monitor.go     # Health check scheduler
│       ├── tcp.go         # TCP health checker
│       ├── http.go        # HTTP/HTTPS health checker
│       ├── ping.go        # ICMP ping checker
│       ├── script.go      # Script checker
│       └── factory.go     # Checker factory
├── pkg/
│   ├── config/            # Configuration loading & validation
│   ├── logger/            # Logging system
│   └── netif/             # Network interface management
└── deployment/            # Deployment files
    ├── go-alived.service  # Systemd service file
    ├── install.sh         # Installation script
    ├── uninstall.sh       # Uninstallation script
    ├── check-env.sh       # Environment check script
    ├── README.md          # Deployment documentation
    └── COMPATIBILITY.md   # Environment compatibility guide
```

## Environment Compatibility

### ✅ Fully Supported
- Physical servers
- KVM/QEMU virtual machines
- Proxmox VE
- VMware ESXi (with promiscuous mode)
- VirtualBox (with bridged network + promiscuous mode)

### ⚠️ Limited Support
- Private cloud (depends on network configuration)
- Docker containers (requires `--privileged` and `--net=host`)
- Kubernetes (requires hostNetwork mode)

### ❌ Not Supported
- AWS EC2 (multicast disabled)
- Aliyun ECS (multicast disabled)
- Azure VM (requires special configuration)
- Google Cloud (multicast disabled by default)

**Why?** Public clouds typically disable multicast protocols (224.0.0.18) at the network virtualization layer.

**Alternative**: Use cloud-native solutions like Elastic IP (AWS), SLB/HaVip (Aliyun), Load Balancer (Azure/GCP).

See [deployment/COMPATIBILITY.md](deployment/COMPATIBILITY.md) for detailed compatibility information.

## Requirements

- Go 1.21+ (for building)
- Linux/macOS with root privileges (for raw sockets and interface management)
- Network interface with IPv4 address
- Multicast support (for VRRP)

## Dependencies

Minimal external dependencies:
- `github.com/vishvananda/netlink` - Network interface management
- `github.com/mdlayher/arp` - ARP packet handling
- `github.com/spf13/cobra` - CLI framework
- `golang.org/x/net/ipv4` - IPv4 raw socket support
- `golang.org/x/net/icmp` - ICMP ping support
- `gopkg.in/yaml.v3` - YAML configuration parsing

## Documentation

- [USAGE.md](USAGE.md) - Detailed usage guide
- [TESTING.md](TESTING.md) - Testing guide
- [deployment/README.md](deployment/README.md) - Deployment guide
- [deployment/COMPATIBILITY.md](deployment/COMPATIBILITY.md) - Environment compatibility
- [roadmap.md](roadmap.md) - Implementation roadmap

## Roadmap

See [roadmap.md](roadmap.md) for detailed implementation plan.

## License

MIT License

## Contributing

Contributions are welcome! Please feel free to submit issues and pull requests.
