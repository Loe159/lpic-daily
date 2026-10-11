#!/usr/bin/env bash
set -euo pipefail
# Runs only in disposable libvirt guest.
test "$(systemd-detect-virt --vm)" = kvm
test -b /dev/vda
test -d /sys/firmware/efi/efivars
state=/var/lib/lpic-daily/uefi-boot-order
test -s "$state/target-order"
if efibootmgr | grep -q '^BootNext:'; then efibootmgr -N; fi
efibootmgr -o "$(cat "$state/target-order")"
