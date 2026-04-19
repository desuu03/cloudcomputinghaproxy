#!/bin/bash
# HAProxy Add Server Script
# Usage: haproxy_add.sh <server_name> <server_ip> <server_port>

NAME=$1
IP=$2
PORT=$3

HAPROXY_CFG="/etc/haproxy/haproxy.cfg"
BACKEND_NAME="images_backend"

echo "[$(date)] Adding server $NAME ($IP:$PORT) to HAProxy"

if [ ! -f "$HAPROXY_CFG" ]; then
    echo "Error: HAProxy config not found at $HAPROXY_CFG"
    exit 1
fi

if grep -q "server $NAME" "$HAPROXY_CFG"; then
    echo "Server $NAME already exists in HAProxy config"
    exit 0
fi

sed -i "/server $BACKEND_NAME/a\        server $NAME $IP:$PORT check inter 2000 rise 2 fall 3" "$HAPROXY_CFG"

if command -v systemctl &> /dev/null; then
    systemctl reload haproxy 2>/dev/null || true
elif command -v service &> /dev/null; then
    service haproxy reload 2>/dev/null || true
fi

echo "Server $NAME added successfully"
exit 0