#!/bin/bash
# HAProxy Remove Server Script
# Usage: haproxy_remove.sh <server_name>

NAME=$1

HAPROXY_CFG="/etc/haproxy/haproxy.cfg"

echo "[$(date)] Removing server $NAME from HAProxy"

if [ ! -f "$HAPROXY_CFG" ]; then
    echo "Error: HAProxy config not found at $HAPROXY_CFG"
    exit 1
fi

sed -i "/server $NAME/d" "$HAPROXY_CFG"

if command -v systemctl &> /dev/null; then
    systemctl reload haproxy 2>/dev/null || true
elif command -v service &> /dev/null; then
    service haproxy reload 2>/dev/null || true
fi

echo "Server $NAME removed successfully"
exit 0