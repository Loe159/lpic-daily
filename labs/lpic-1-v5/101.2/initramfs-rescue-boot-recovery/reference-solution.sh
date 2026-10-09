#!/usr/bin/env bash
set -euo pipefail
k=$(cat /var/lib/lpic-initramfs-rescue/kernel-release)
id=$(cat /var/lib/lpic-initramfs-rescue/entry-id)
dracut --force --hostonly --include /etc/lpic-initramfs/payload /etc/lpic-initramfs/payload "/boot/initramfs-lpic-rescue-$k.img" "$k"
lsinitrd -f /etc/lpic-initramfs/payload "/boot/initramfs-lpic-rescue-$k.img" | grep -Fxq 'rescue-profile=storage-audit'
grub2-set-default "$id"
# :reboot est indispensable pour le second contrôle.
