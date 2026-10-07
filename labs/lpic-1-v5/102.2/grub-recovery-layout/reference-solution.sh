#!/usr/bin/env bash
set -euo pipefail
parted -s /dev/vdb mklabel msdos
parted -s /dev/vdb mkpart primary ext4 1MiB 100%
cat >> /etc/grub.d/40_custom <<'EOF'
menuentry 'LPIC Recovery' {
    echo 'LPIC recovery entry'
}
EOF
grub2-mkconfig -o /boot/grub2/grub.cfg
mkdir -p /root/lpic-grub
grub2-install --version > /root/lpic-grub/version.txt
{
  test -f /boot/grub2/grub.cfg && echo grub2-config=present
  if test -e /boot/grub/menu.lst || test -e /boot/grub2/menu.lst; then echo legacy-menu=present; else echo legacy-menu=absent; fi
} > /root/lpic-grub/layout.txt
