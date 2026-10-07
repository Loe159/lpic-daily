#!/usr/bin/env bash
set -euo pipefail
mkdir -p /root/lpic-virt
systemd-detect-virt > /root/lpic-virt/virt-type.txt
lsblk -dn -o NAME,TYPE > /root/lpic-virt/block.txt
ip -o link show > /root/lpic-virt/network.txt
cat /etc/machine-id > /root/lpic-virt/machine-id.txt
for key in /etc/ssh/ssh_host_*_key.pub; do ssh-keygen -lf "$key" -E sha256; done > /root/lpic-virt/ssh-hostkeys.txt
systemctl is-active qemu-guest-agent > /root/lpic-virt/guest-agent.txt
{ test -f /etc/cloud/cloud.cfg && printf 'cloud-config:present\n'; cloud-init status 2>&1 || true; } > /root/lpic-virt/cloud.txt
