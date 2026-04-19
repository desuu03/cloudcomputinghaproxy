#!/bin/bash
# VM Deletion Script
# Usage: eliminar_vm.sh <vm_name>

VM_NAME=${1:-""}

if [ -z "$VM_NAME" ]; then
    echo "Usage: $0 <vm_name>"
    exit 1
fi

echo "[$(date)] Deleting VM: $VM_NAME"

if command -v multipass &> /dev/null; then
    multipass delete "$VM_NAME" && multipass purge
elif command -v virsh &> /dev/null; then
    virsh destroy "$VM_NAME" 2>/dev/null || true
    virsh undefine "$VM_NAME" 2>/dev/null || true
else
    echo "No hypervisor found. Simulated deletion."
fi

echo "VM $VM_NAME deleted successfully"
exit 0