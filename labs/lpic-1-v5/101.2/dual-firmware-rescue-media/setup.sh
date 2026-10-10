#!/usr/bin/env bash
set -euo pipefail
for cmd in parted mkfs.ext4 mkfs.vfat sfdisk blkid grub2-install; do command -v "$cmd" >/dev/null; done
test -d /sys/firmware/efi
test -b /dev/vdb && test -b /dev/vdc
install -d -m 0755 /mnt/lpic-bios-rescue /mnt/lpic-efi-rescue /var/lib/lpic-dual-firmware
parted --script /dev/vdb mklabel msdos mkpart primary ext4 1MiB 200MiB
parted --script /dev/vdc mklabel gpt mkpart primary fat32 1MiB 200MiB set 1 esp on
udevadm settle
test -b /dev/vdb1 && test -b /dev/vdc1
mkfs.ext4 -F -q -L LPICBIOS /dev/vdb1
mkfs.vfat -F32 -n LPICUEFI /dev/vdc1
mount /dev/vdb1 /mnt/lpic-bios-rescue
mount /dev/vdc1 /mnt/lpic-efi-rescue
install -d -m 0755 /mnt/lpic-bios-rescue/boot/grub2
grub2-install --target=i386-pc --boot-directory=/mnt/lpic-bios-rescue/boot --recheck /dev/vdb
test -s /mnt/lpic-bios-rescue/boot/grub2/i386-pc/core.img
printf 'legacy-inventory=retain-35\n' > /mnt/lpic-bios-rescue/manifest.txt
printf 'efi-inventory=retain-35\n' > /mnt/lpic-efi-rescue/manifest.txt
mountpoint -q /boot/efi || mount /boot/efi
test -d /boot/efi/EFI
source_efi=$(find /boot/efi/EFI -type f -iname '*.efi' -print -quit)
test -n "$source_efi" && test -s "$source_efi"
install -d -m 0755 /mnt/lpic-efi-rescue/EFI/BOOT
cp "$source_efi" /mnt/lpic-efi-rescue/EFI/BOOT/BOOTX64.EFI
sha256sum /mnt/lpic-efi-rescue/EFI/BOOT/BOOTX64.EFI | awk '{print $1}' > /var/lib/lpic-dual-firmware/efi-sha256
sfdisk -d /dev/vdb | sed -n '/^\/dev\/vdb1[[:space:]]/p' > /var/lib/lpic-dual-firmware/legacy-partition
test -s /var/lib/lpic-dual-firmware/legacy-partition
sync
parted --script /dev/vdc set 1 msftdata on
udevadm settle
test "$(blkid -s PART_ENTRY_TYPE -o value /dev/vdc1 | tr '[:upper:]' '[:lower:]')" != c12a7328-f81f-11d2-ba4b-00a0c93ec93b
