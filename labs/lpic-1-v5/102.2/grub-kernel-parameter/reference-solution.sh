#!/usr/bin/env bash
set -euo pipefail
config=/etc/default/grub
for token in loglevel=7 systemd.show_status=yes; do
  if grep -q "^GRUB_CMDLINE_LINUX=" "$config"; then
    if ! grep -q "$token" "$config"; then
      sed -i "/^GRUB_CMDLINE_LINUX=/ s/\"$/ $token\"/" "$config"
    fi
  else
    printf 'GRUB_CMDLINE_LINUX="%s"\n' "$token" >> "$config"
  fi
done
grubby --update-kernel=ALL --args="loglevel=7 systemd.show_status=yes"
grub2-mkconfig -o /boot/grub2/grub.cfg
# :reboot is a separate learner action; the VM must start again for /proc/cmdline to change.
