#!/usr/bin/env bash
set -euo pipefail
test -d /sys/firmware/efi
test -b /dev/vdb
mkdir -p /mnt/lpic-rescue-esp
parted --script /dev/vdb mklabel gpt
parted --script /dev/vdb mkpart primary fat32 1MiB 100%
partprobe /dev/vdb
for _ in $(seq 1 100); do test -b /dev/vdb1 && break; sleep 0.1; done
test -b /dev/vdb1
mkfs.vfat -F 32 -n LPICRESCUE /dev/vdb1 >/dev/null
parted --script /dev/vdb set 1 esp off
mount /dev/vdb1 /mnt/lpic-rescue-esp
install -d -m 0755 /mnt/lpic-rescue-esp/EFI/BOOT
source_efi=
for candidate in /boot/efi/EFI/BOOT/BOOTX64.EFI /boot/efi/EFI/fedora/shimx64.efi /boot/efi/EFI/fedora/grubx64.efi; do
  if test -f "$candidate"; then source_efi="$candidate"; break; fi
done
if test -z "$source_efi"; then
  source_efi=$(find /boot/efi -type f -iname '*.efi' -print -quit)
fi
test -n "$source_efi" && file -b "$source_efi" | grep -Eq '^PE32'
cp "$source_efi" /mnt/lpic-rescue-esp/EFI/BOOT/BOOTX64.EFI
printf 'recovery=retain-efi-loader\n' > /mnt/lpic-rescue-esp/manifest.txt
sha256sum /mnt/lpic-rescue-esp/EFI/BOOT/BOOTX64.EFI > /root/lpic-rescue-efi.sha256
sync
umount /mnt/lpic-rescue-esp
test "$(lsblk -dn -o PARTTYPE /dev/vdb1 | tr '[:upper:]' '[:lower:]')" != c12a7328-f81f-11d2-ba4b-00a0c93ec93b
