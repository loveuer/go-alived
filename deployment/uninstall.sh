#!/bin/bash
set -e

echo "=== Uninstalling go-alived ==="

if [ "$EUID" -ne 0 ]; then 
    echo "Please run as root (use sudo)"
    exit 1
fi

BINARY_PATH="/usr/local/bin/go-alived"
CONFIG_DIR="/etc/go-alived"
SERVICE_FILE="/etc/systemd/system/go-alived.service"

if systemctl is-active --quiet go-alived; then
    echo "1. Stopping service..."
    systemctl stop go-alived
    echo "   ✓ Service stopped"
fi

if systemctl is-enabled --quiet go-alived 2>/dev/null; then
    echo "2. Disabling service..."
    systemctl disable go-alived
    echo "   ✓ Service disabled"
fi

if [ -f "${SERVICE_FILE}" ]; then
    echo "3. Removing service file..."
    rm ${SERVICE_FILE}
    systemctl daemon-reload
    echo "   ✓ Service file removed"
fi

if [ -f "${BINARY_PATH}" ]; then
    echo "4. Removing binary..."
    rm ${BINARY_PATH}
    echo "   ✓ Binary removed"
fi

echo ""
read -p "Do you want to remove configuration directory ${CONFIG_DIR}? (y/N) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    if [ -d "${CONFIG_DIR}" ]; then
        rm -rf ${CONFIG_DIR}
        echo "   ✓ Configuration removed"
    fi
else
    echo "   ⚠ Configuration kept at ${CONFIG_DIR}"
fi

echo ""
echo "=== Uninstallation complete ==="
