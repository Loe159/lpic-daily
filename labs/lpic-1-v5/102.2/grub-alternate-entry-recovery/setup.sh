#!/usr/bin/env bash
set -euo pipefail
command -v grub2-set-default >/dev/null
install -d -m 0755 /var/lib/lpic-grub-recovery
cat /proc/sys/kernel/random/boot_id > /var/lib/lpic-grub-recovery/initial-boot-id
uname -r > /var/lib/lpic-grub-recovery/kernel-release
source=''
for entry in /boot/loader/entries/*.conf; do
  test -f "$entry" || continue
  if grep -Fq "/vmlinuz-$(uname -r)" "$entry"; then source="$entry"; break; fi
done
test -n "$source"
id="lpic-archive-rescue-$(uname -r)"
cp "$source" "/boot/loader/entries/$id.conf"
sed -i -e 's/^title .*/title LPIC archive diagnostic recovery/' -e '/^options /s/$/ lpic.archive_rescue=1/' "/boot/loader/entries/$id.conf"
grep -q '^options .*lpic.archive_rescue=1' "/boot/loader/entries/$id.conf"
printf '%s\n' "$id" > /var/lib/lpic-grub-recovery/entry-id
! tr ' ' '\n' </proc/cmdline | grep -Fxq lpic.archive_rescue=1
