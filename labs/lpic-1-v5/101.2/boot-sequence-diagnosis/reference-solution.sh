#!/usr/bin/env bash
set -euo pipefail
token=lpic_daily_stage=1012
if ! grep -qw "$token" /proc/cmdline; then
  if ! grep -q "$token" /etc/default/grub; then
    sed -i '/^GRUB_CMDLINE_LINUX=/ { /lpic_daily_stage=1012/! s/"$/ lpic_daily_stage=1012"/ }' /etc/default/grub
  fi
  grub2-mkconfig -o /boot/grub2/grub.cfg
  printf 'Configuration prête. Utilise :reboot puis relance les commandes de collecte.\n'
  exit 0
fi
mkdir -p /root/lpic-boot
cat /proc/cmdline > /root/lpic-boot/cmdline.txt
if test -d /sys/firmware/efi; then printf 'UEFI\n'; else printf 'BIOS\n'; fi > /root/lpic-boot/firmware.txt
lsinitrd "/boot/initramfs-$(uname -r).img" > /root/lpic-boot/initramfs.txt
ps -p 1 -o comm= | xargs > /root/lpic-boot/pid1.txt
readlink -f /sbin/init > /root/lpic-boot/init-link.txt
journalctl -b -k --no-pager > /root/lpic-boot/kernel-journal.txt
