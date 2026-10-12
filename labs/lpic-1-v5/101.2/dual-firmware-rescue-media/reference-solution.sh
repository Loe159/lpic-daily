#!/usr/bin/env bash
set -euo pipefail
parted --script /dev/vdc set 1 esp on
udevadm settle
test "$(blkid -s PART_ENTRY_TYPE -o value /dev/vdc1 | tr '[:upper:]' '[:lower:]')" = c12a7328-f81f-11d2-ba4b-00a0c93ec93b
test -s /mnt/lpic-efi-rescue/EFI/BOOT/BOOTX64.EFI
