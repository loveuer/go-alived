package cmd

import (
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

const (
	defaultBinaryPath  = "/usr/local/bin/go-alived"
	defaultConfigDir   = "/etc/go-alived"
	defaultConfigFile  = "/etc/go-alived/config.yaml"
	systemdServicePath = "/etc/systemd/system/go-alived.service"
	initdScriptPath    = "/etc/init.d/go-alived"
)

var (
	installMethod string
)

var installCmd = &cobra.Command{
	Use:     "install",
	Aliases: []string{"i"},
	Short:   "Install go-alived as a system service",
	Long: `Install go-alived binary and configuration files to system paths.

Supported installation methods:
  - systemd: Install as a systemd service (default, recommended for modern Linux)
  - service: Install as a SysV init.d service (for older Linux distributions)

Examples:
  sudo go-alived install
  sudo go-alived install --method systemd
  sudo go-alived i -m service`,
	Run: runInstall,
}

func init() {
	rootCmd.AddCommand(installCmd)

	installCmd.Flags().StringVarP(&installMethod, "method", "m", "systemd",
		"installation method: systemd, service")
}

func runInstall(cmd *cobra.Command, args []string) {
	// Check root privileges
	if os.Geteuid() != 0 {
		fmt.Println("Error: This command requires root privileges")
		fmt.Println("Please run with: sudo go-alived install")
		os.Exit(1)
	}

	// Validate method
	method := strings.ToLower(installMethod)
	if method != "systemd" && method != "service" {
		fmt.Printf("Error: Invalid installation method '%s'\n", installMethod)
		fmt.Println("Supported methods: systemd, service")
		os.Exit(1)
	}

	fmt.Println("=== Go-Alived Installation ===")
	fmt.Println()

	const totalSteps = 3

	// Step 1: Copy binary
	if err := installBinary(1, totalSteps); err != nil {
		fmt.Printf("Error installing binary: %v\n", err)
		os.Exit(1)
	}

	// Step 2: Create config directory and file
	configCreated, err := installConfig(2, totalSteps)
	if err != nil {
		fmt.Printf("Error installing config: %v\n", err)
		os.Exit(1)
	}

	// Step 3: Install service script
	if err := installServiceScript(3, totalSteps, method); err != nil {
		fmt.Printf("Error installing service script: %v\n", err)
		os.Exit(1)
	}

	// Print completion message
	printCompletionMessage(method, configCreated)
}

func installBinary(step, total int) error {
	fmt.Printf("[%d/%d] Installing binary... ", step, total)

	// Get current executable path
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Resolve symlinks
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("failed to resolve symlinks: %w", err)
	}

	// Check if already installed at target path
	if execPath == defaultBinaryPath {
		fmt.Println("already installed")
		return nil
	}

	// Open source file
	src, err := os.Open(execPath)
	if err != nil {
		return fmt.Errorf("failed to open source binary: %w", err)
	}
	defer src.Close()

	// Create destination file
	dst, err := os.OpenFile(defaultBinaryPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return fmt.Errorf("failed to create destination binary: %w", err)
	}
	defer dst.Close()

	// Copy binary
	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("failed to copy binary: %w", err)
	}

	fmt.Printf("done (%s)\n", defaultBinaryPath)
	return nil
}

func installConfig(step, total int) (bool, error) {
	fmt.Printf("[%d/%d] Setting up configuration... ", step, total)

	// Create config directory
	if err := os.MkdirAll(defaultConfigDir, 0755); err != nil {
		return false, fmt.Errorf("failed to create config directory: %w", err)
	}

	// Check if config file already exists
	if _, err := os.Stat(defaultConfigFile); err == nil {
		fmt.Println("config already exists")
		return false, nil
	}

	// Generate config content
	configContent := generateDefaultConfig()

	// Write config file
	if err := os.WriteFile(defaultConfigFile, []byte(configContent), 0644); err != nil {
		return false, fmt.Errorf("failed to write config file: %w", err)
	}

	fmt.Printf("done (%s)\n", defaultConfigFile)
	return true, nil
}

func installServiceScript(step, total int, method string) error {
	switch method {
	case "systemd":
		return installSystemdService(step, total)
	case "service":
		return installInitdScript(step, total)
	default:
		return fmt.Errorf("unsupported method: %s", method)
	}
}

func installSystemdService(step, total int) error {
	fmt.Printf("[%d/%d] Installing systemd service... ", step, total)

	serviceContent := generateSystemdService()

	if err := os.WriteFile(systemdServicePath, []byte(serviceContent), 0644); err != nil {
		return fmt.Errorf("failed to write service file: %w", err)
	}

	fmt.Printf("done (%s)\n", systemdServicePath)
	return nil
}

func installInitdScript(step, total int) error {
	fmt.Printf("[%d/%d] Installing init.d script... ", step, total)

	scriptContent := generateInitdScript()

	if err := os.WriteFile(initdScriptPath, []byte(scriptContent), 0755); err != nil {
		return fmt.Errorf("failed to write init.d script: %w", err)
	}

	fmt.Printf("done (%s)\n", initdScriptPath)
	return nil
}

func generateDefaultConfig() string {
	// Auto-detect network interface
	iface := detectNetworkInterface()
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "node1"
	}

	return fmt.Sprintf(`# Go-Alived Configuration
# Generated by: go-alived install
# Documentation: https://github.com/loveuer/go-alived

global:
  router_id: "%s"

vrrp_instances:
  - name: "VI_1"
    interface: "%s"
    state: "BACKUP"
    virtual_router_id: 51
    priority: 100
    advert_interval: 1
    auth_type: "PASS"
    auth_pass: "changeme"    # TODO: Change this password
    virtual_ips:
      - "192.168.1.100/24"   # TODO: Change to your VIP

# Optional: Health checkers
# health_checkers:
#   - name: "check_nginx"
#     type: "tcp"
#     interval: 3s
#     timeout: 2s
#     rise: 3
#     fall: 2
#     config:
#       host: "127.0.0.1"
#       port: 80
`, hostname, iface)
}

func generateSystemdService() string {
	return `[Unit]
Description=Go-Alived - VRRP High Availability Service
Documentation=https://github.com/loveuer/go-alived
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=root
Group=root

ExecStart=/usr/local/bin/go-alived run --config /etc/go-alived/config.yaml
ExecReload=/bin/kill -HUP $MAINPID

Restart=on-failure
RestartSec=5s

StandardOutput=journal
StandardError=journal
SyslogIdentifier=go-alived

# Security settings
NoNewPrivileges=false
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/etc/go-alived

# Resource limits
LimitNOFILE=65535
LimitNPROC=512

# Capabilities required for VRRP operations
AmbientCapabilities=CAP_NET_ADMIN CAP_NET_RAW CAP_NET_BIND_SERVICE
CapabilityBoundingSet=CAP_NET_ADMIN CAP_NET_RAW CAP_NET_BIND_SERVICE

[Install]
WantedBy=multi-user.target
`
}

func generateInitdScript() string {
	return `#!/bin/sh
### BEGIN INIT INFO
# Provides:          go-alived
# Required-Start:    $network $remote_fs $syslog
# Required-Stop:     $network $remote_fs $syslog
# Default-Start:     2 3 4 5
# Default-Stop:      0 1 6
# Short-Description: Go-Alived VRRP High Availability Service
# Description:       Lightweight VRRP implementation for IP high availability
### END INIT INFO

NAME="go-alived"
DAEMON="/usr/local/bin/go-alived"
DAEMON_ARGS="run --config /etc/go-alived/config.yaml"
PIDFILE="/var/run/${NAME}.pid"
LOGFILE="/var/log/${NAME}.log"

[ -x "$DAEMON" ] || exit 5

start() {
    if [ -f "$PIDFILE" ] && kill -0 "$(cat "$PIDFILE")" 2>/dev/null; then
        echo "$NAME is already running"
        return 1
    fi
    echo -n "Starting $NAME... "
    nohup $DAEMON $DAEMON_ARGS >> "$LOGFILE" 2>&1 &
    echo $! > "$PIDFILE"
    echo "done (PID: $(cat "$PIDFILE"))"
}

stop() {
    if [ ! -f "$PIDFILE" ] || ! kill -0 "$(cat "$PIDFILE")" 2>/dev/null; then
        echo "$NAME is not running"
        return 1
    fi
    echo -n "Stopping $NAME... "
    kill "$(cat "$PIDFILE")"
    rm -f "$PIDFILE"
    echo "done"
}

restart() {
    stop
    sleep 1
    start
}

reload() {
    if [ ! -f "$PIDFILE" ] || ! kill -0 "$(cat "$PIDFILE")" 2>/dev/null; then
        echo "$NAME is not running"
        return 1
    fi
    echo -n "Reloading $NAME configuration... "
    kill -HUP "$(cat "$PIDFILE")"
    echo "done"
}

status() {
    if [ -f "$PIDFILE" ] && kill -0 "$(cat "$PIDFILE")" 2>/dev/null; then
        echo "$NAME is running (PID: $(cat "$PIDFILE"))"
    else
        echo "$NAME is not running"
        [ -f "$PIDFILE" ] && rm -f "$PIDFILE"
        return 1
    fi
}

case "$1" in
    start)   start   ;;
    stop)    stop    ;;
    restart) restart ;;
    reload)  reload  ;;
    status)  status  ;;
    *)
        echo "Usage: $0 {start|stop|restart|reload|status}"
        exit 2
        ;;
esac

exit $?
`
}

func detectNetworkInterface() string {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "eth0"
	}

	for _, iface := range interfaces {
		// Skip loopback and down interfaces
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		if iface.Flags&net.FlagUp == 0 {
			continue
		}

		// Check if interface has IPv4 address
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			if ipNet, ok := addr.(*net.IPNet); ok {
				if ipv4 := ipNet.IP.To4(); ipv4 != nil && !ipv4.IsLoopback() {
					return iface.Name
				}
			}
		}
	}

	return "eth0"
}

func printCompletionMessage(method string, configCreated bool) {
	fmt.Println()
	fmt.Println("=== Installation Complete ===")
	fmt.Println()

	// Installed files summary
	fmt.Println(">>> Installed Files:")
	fmt.Printf("    Binary:  %s\n", defaultBinaryPath)
	fmt.Printf("    Config:  %s\n", defaultConfigFile)
	if method == "systemd" {
		fmt.Printf("    Service: %s\n", systemdServicePath)
	} else {
		fmt.Printf("    Service: %s\n", initdScriptPath)
	}
	fmt.Println()

	// What needs to be modified
	fmt.Println(">>> Configuration Required:")
	fmt.Printf("    Edit: %s\n", defaultConfigFile)
	fmt.Println()
	if configCreated {
		fmt.Println("    Modify the following settings:")
		fmt.Println("    - auth_pass:    Change 'changeme' to a secure password")
		fmt.Println("    - virtual_ips:  Set your Virtual IP address(es)")
		fmt.Println("    - interface:    Verify the network interface is correct")
		fmt.Println("    - priority:     Adjust based on node role (higher = more likely master)")
	} else {
		fmt.Println("    Review your existing configuration")
	}
	fmt.Println()

	// How to start
	fmt.Println(">>> Next Steps:")
	if method == "systemd" {
		fmt.Println("    1. Edit configuration:")
		fmt.Printf("       sudo vim %s\n", defaultConfigFile)
		fmt.Println()
		fmt.Println("    2. Reload systemd and start service:")
		fmt.Println("       sudo systemctl daemon-reload")
		fmt.Println("       sudo systemctl enable go-alived")
		fmt.Println("       sudo systemctl start go-alived")
		fmt.Println()
		fmt.Println("    3. Check service status:")
		fmt.Println("       sudo systemctl status go-alived")
		fmt.Println("       sudo journalctl -u go-alived -f")
	} else {
		fmt.Println("    1. Edit configuration:")
		fmt.Printf("       sudo vim %s\n", defaultConfigFile)
		fmt.Println()
		fmt.Println("    2. Start service:")
		fmt.Printf("       sudo %s start\n", initdScriptPath)
		fmt.Println()
		fmt.Println("    3. Enable on boot (Debian/Ubuntu):")
		fmt.Println("       sudo update-rc.d go-alived defaults")
		fmt.Println()
		fmt.Println("    4. Check service status:")
		fmt.Printf("       sudo %s status\n", initdScriptPath)
		fmt.Printf("       tail -f /var/log/go-alived.log\n")
	}
	fmt.Println()

	// Test environment
	fmt.Println(">>> Test Environment (Optional):")
	fmt.Printf("    sudo %s test\n", defaultBinaryPath)
	fmt.Println()
}
