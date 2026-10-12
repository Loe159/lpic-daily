#!/usr/bin/env bash
set -euo pipefail
command -v grub2-editenv >/dev/null
command -v grub2-set-default >/dev/null
state=/var/lib/lpic-grub-legacy
install -d -m 0755 "$state"
k="$(uname -r)"
uname -r > "$state/kernel-release"
cat /proc/sys/kernel/random/boot_id > "$state/initial-boot-id"
source=''
for candidate in /boot/loader/entries/*.conf; do
  test -f "$candidate" || continue
  if grep -Fq "/vmlinuz-$k" "$candidate"; then
    source="$candidate"
    break
  fi
done
test -n "$source"
grep -q '^linux ' "$source"
grep -q '^initrd ' "$source"
grep -q '^options ' "$source"
printf '%s\n' "$source" > "$state/source-entry"
id="lpic-legacy-import-$k"
printf '%s\n' "$id" > "$state/import-id"
test ! -e "/boot/loader/entries/$id.conf"
cat > "$state/menu.lst" <<EOF
default=0
timeout=5
title LPIC recovery (GRUB Legacy archive)
root (hd0,0)
kernel /vmlinuz-$k ro lpic.legacy_import=1
initrd /initramfs-$k.img
EOF
sha256sum "$source" "$state/menu.lst" > "$state/preserved.sha256"
grub2-set-default 0
! tr ' ' '\n' </proc/cmdline | grep -Fxq lpic.legacy_import=1
