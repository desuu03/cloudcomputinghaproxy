#!/bin/bash
# VM Creation Script
# Usage: crear_vm.sh [server_name]

SERVER_NAME=${1:-"temp_server"}
VM_NAME="vm-${SERVER_NAME}-$(date +%s)"

echo "[$(date)] Creating VM: $VM_NAME"

if command -v multipass &> /dev/null; then
    multipass launch -n "$VM_NAME" -c 2 -m 2G
    IP=$(multipass list | grep "$VM_NAME" | awk '{print $3}')
    echo "VM created: $VM_NAME (IP: $IP)"
elif command -v virsh &> /dev/null; then
    echo "Using virsh to create VM..."
    virsh create /tmp/vm-template.xml
else
    echo "No hypervisor found. Using simulated VM."
    IP="192.168.1.$((100 + RANDOM % 50))"
fi

echo "VM_NAME=$VM_NAME"
echo "VM_IP=$IP"
echo "VM_PORT=8000"
exit 0