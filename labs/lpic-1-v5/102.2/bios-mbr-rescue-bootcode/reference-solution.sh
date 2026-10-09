#!/usr/bin/env bash
set -euo pipefail
mountpoint -q /mnt/lpic-bios-rescue
grub2-install --target=i386-pc --boot-directory=/mnt/lpic-bios-rescue/boot --recheck /dev/vdb
test -s /mnt/lpic-bios-rescue/boot/grub2/grub.cfg
