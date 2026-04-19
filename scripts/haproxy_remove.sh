#!/bin/bash
# HAProxy Remove Server Script
# Usage: haproxy_remove.sh <server_name> <ip> <port>

NAME=$1
IP=$2
PORT=$3

HAPROXY_CFG="/etc/haproxy/haproxy.cfg"

echo "[$(date)] Removing server $NAME from HAProxy"

if [ -f "$HAPROXY_CFG" ]; then
    sed -i "/server $NAME/d" "$HAPROXY_CFG"
    
    # Recargar HAProxy
    if command -v systemctl &> /dev/null; then
        sudo systemctl reload haproxy 2>/dev/null || true
    fi
fi

echo "Server $NAME removed"
exit 0