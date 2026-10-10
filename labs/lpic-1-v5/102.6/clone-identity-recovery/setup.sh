#!/usr/bin/env bash
set -euo pipefail
mkdir -p /workspace/vms/{node-a,node-b}/etc/ssh /workspace/vms/{node-a,node-b}/opt/app
cat /proc/sys/kernel/random/uuid | tr -d '-' > /workspace/vms/node-a/etc/machine-id
ssh-keygen -q -t ed25519 -N '' -f /workspace/vms/node-a/etc/ssh/ssh_host_ed25519_key
cp -a /workspace/vms/node-a/etc/. /workspace/vms/node-b/etc/
printf 'region=eu-west\n' | tee /workspace/vms/node-a/opt/app/config.ini /workspace/vms/node-b/opt/app/config.ini >/dev/null
