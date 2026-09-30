#!/usr/bin/env bash
set -euo pipefail

config=/etc/default/grub
token=lpic_daily_boot=phase2

if ! grep -q "$token" "$config"; then
  if grep -q '^GRUB_CMDLINE_LINUX=' "$config"; then
    sed -i '/^GRUB_CMDLINE_LINUX=/ { /lpic_daily_boot=phase2/! s/"$/ lpic_daily_boot=phase2"/ }' "$config"
  else
    printf 'GRUB_CMDLINE_LINUX="%s"\n' "$token" >> "$config"
  fi
fi

grub2-mkconfig -o /boot/grub2/grub.cfg
