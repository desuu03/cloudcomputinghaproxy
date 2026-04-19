#!/bin/bash
# HAProxy Add Server Script
# Usage: haproxy_add.sh <server_name> <server_ip> <server_port>

NAME=$1
IP=$2
PORT=$3

HAPROXY_CFG="/etc/haproxy/haproxy.cfg"
BACKEND_NAME="images_backend"

echo "[$(date)] Adding server $NAME ($IP:$PORT) to HAProxy"

# Crear config si no existe
if [ ! -f "$HAPROXY_CFG" ]; then
    echo "Creating HAProxy config at $HAPROXY_CFG"
    sudo mkdir -p /etc/haproxy
    cat > "$HAPROXY_CFG" << EOF
global
    log /dev/log    local0
    maxconn 4096

defaults
    log     global
    mode    http
    timeout connect 5000ms
    timeout client  50000ms
    timeout server 50000ms

frontend images_frontend
    bind *:80
    bind *:8000
    default_backend images_backend

backend images_backend
    balance roundrobin
    option httpchk GET /
    server primary 127.0.0.1:8000 check inter 2000 rise 2 fall 3
EOF
fi

# Agregar server si no existe
if ! grep -q "server $NAME" "$HAPROXY_CFG"; then
    echo "server $NAME $IP:$PORT check inter 2000 rise 2 fall 3" >> "$HAPROXY_CFG"
fi

# Recargar HAProxy
if command -v systemctl &> /dev/null; then
    sudo systemctl reload haproxy 2>/dev/null || sudo systemctl start haproxy
elif command -v haproxy &> /dev/null; then
    sudo haproxy -f "$HAPROXY_CFG" -sf $(pidof haproxy) 2>/dev/null || true
fi

echo "Server $NAME added to $HAPROXY_CFG"
exit 0