#!/usr/bin/env bash
set -euo pipefail
for cmd in grub2-install parted sfdisk mkfs.ext4; do command -v "$cmd" >/dev/null; done
test -b /dev/vdb
install -d -m 0755 /mnt/lpic-bios-rescue /var/lib/lpic-mbr-rescue
parted --script /dev/vdb mklabel msdos mkpart primary ext4 1MiB 250MiB set 1 boot on
udevadm settle
test -b /dev/vdb1
mkfs.ext4 -F -q -L LPICBIOS /dev/vdb1
mount /dev/vdb1 /mnt/lpic-bios-rescue
printf 'recovery-index=preserve-93\n' > /mnt/lpic-bios-rescue/index.txt
install -d -m 0755 /mnt/lpic-bios-rescue/boot/grub2
cat > /mnt/lpic-bios-rescue/boot/grub2/grub.cfg <<'EOF'
set timeout=5
menuentry 'LPIC BIOS rescue inventory' {
    echo 'Use the inventory before selecting a recovery kernel'
}
EOF
sfdisk -d /dev/vdb | sed -n '/^\/dev\/vdb1[[:space:]]/p' > /var/lib/lpic-mbr-rescue/partition-snapshot
test -s /var/lib/lpic-mbr-rescue/partition-snapshot
dd if=/dev/zero of=/dev/vdb bs=440 count=1 conv=notrunc status=none
sync
! dd if=/dev/vdb bs=440 count=1 status=none | strings | grep -Fq GRUB
