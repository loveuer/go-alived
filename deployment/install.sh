#!/bin/bash
set -e

echo "=== Installing go-alived ==="

if [ "$EUID" -ne 0 ]; then 
    echo "Please run as root (use sudo)"
    exit 1
fi

BINARY_PATH="/usr/local/bin/go-alived"
CONFIG_DIR="/etc/go-alived"
SERVICE_FILE="/etc/systemd/system/go-alived.service"

echo "1. Installing binary to ${BINARY_PATH}..."
if [ ! -f "go-alived" ]; then
    echo "Error: go-alived binary not found. Please run 'go build' first."
    exit 1
fi
cp go-alived ${BINARY_PATH}
chmod +x ${BINARY_PATH}
echo "   ✓ Binary installed"

echo "2. Creating configuration directory ${CONFIG_DIR}..."
mkdir -p ${CONFIG_DIR}
mkdir -p ${CONFIG_DIR}/scripts
echo "   ✓ Directories created"

if [ ! -f "${CONFIG_DIR}/config.yaml" ]; then
    echo "3. Installing example configuration..."
    cp config.example.yaml ${CONFIG_DIR}/config.yaml
    echo "   ✓ Configuration installed to ${CONFIG_DIR}/config.yaml"
    echo "   ⚠ Please edit ${CONFIG_DIR}/config.yaml before starting the service"
else
    echo "3. Configuration already exists at ${CONFIG_DIR}/config.yaml"
    echo "   ⚠ Skipping configuration installation"
fi

echo "4. Installing systemd service..."
cp deployment/go-alived.service ${SERVICE_FILE}
systemctl daemon-reload
echo "   ✓ Service installed"

echo ""
echo "=== Installation complete ==="
echo ""
echo "Next steps:"
echo "  1. Edit configuration: vim ${CONFIG_DIR}/config.yaml"
echo "  2. Start service:      systemctl start go-alived"
echo "  3. Check status:       systemctl status go-alived"
echo "  4. View logs:          journalctl -u go-alived -f"
echo "  5. Enable autostart:   systemctl enable go-alived"
echo ""
