#!/usr/bin/env bash
set -euo pipefail
cat /proc/sys/kernel/random/uuid | tr -d '-' > /workspace/vms/node-b/etc/machine-id
rm -f /workspace/vms/node-b/etc/ssh/ssh_host_ed25519_key{,.pub}
ssh-keygen -q -t ed25519 -N '' -f /workspace/vms/node-b/etc/ssh/ssh_host_ed25519_key
