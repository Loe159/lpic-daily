#!/usr/bin/env bash
set -euo pipefail
command -v grub2-editenv >/dev/null
command -v grub2-mkconfig >/dev/null
test -d /boot/loader/entries
test -e /boot/grub2/grubenv
test -s /boot/grub2/grub.cfg
install -d -m 0755 /var/lib/lpic-grub-console
cat /proc/sys/kernel/random/boot_id > /var/lib/lpic-grub-console/initial-boot-id

# Give the learner a reliable window to reach serial GRUB without racing QGA.
sed -i '/^GRUB_TIMEOUT_STYLE=/d; /^GRUB_TIMEOUT=/d' /etc/default/grub
printf 'GRUB_TIMEOUT_STYLE=menu\nGRUB_TIMEOUT=90\n' >> /etc/default/grub
grub2-editenv /boot/grub2/grubenv unset menu_auto_hide
grub2-editenv /boot/grub2/grubenv unset lpic_grub_cli
grub2-mkconfig -o /boot/grub2/grub.cfg
test -s /boot/grub2/grub.cfg
test -f /boot/loader/entries/*.conf || ls /boot/loader/entries/*.conf >/dev/null

# Protect the current bootloader and BLS entries against "fixes" that
# persistently inject the requested temporary kernel argument.
sha256sum /etc/default/grub /boot/grub2/grub.cfg /boot/loader/entries/*.conf \
  > /var/lib/lpic-grub-console/preserved-boot-config.sha256
! tr ' ' '\n' </proc/cmdline | grep -Fxq lpic.console_probe=1
! grub2-editenv /boot/grub2/grubenv list | grep -Fxq 'lpic_grub_cli=verified'
