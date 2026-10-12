#!/usr/bin/env bash
set -euo pipefail
for program in dracut lsinitrd grub2-set-default sha256sum; do command -v "$program" >/dev/null; done
test -d /sys/firmware/efi
install -d -m 0755 /var/lib/lpic-initramfs-rescue /etc/lpic-initramfs
k=$(uname -r)
printf '%s\n' "$k" > /var/lib/lpic-initramfs-rescue/kernel-release
cat /proc/sys/kernel/random/boot_id > /var/lib/lpic-initramfs-rescue/initial-boot-id
source=''
for entry in /boot/loader/entries/*.conf; do
    test -f "$entry" || continue
    if grep -Fq "/vmlinuz-$k" "$entry"; then source="$entry"; break; fi
done
test -n "$source"
grep -q '^initrd ' "$source"
grep -q '^options ' "$source"
printf '%s\n' "$source" > /var/lib/lpic-initramfs-rescue/original-entry
sha256sum "$source" | awk '{print $1}' > /var/lib/lpic-initramfs-rescue/original-sha256
id="lpic-initramfs-rescue-$k"
printf '%s\n' "$id" > /var/lib/lpic-initramfs-rescue/entry-id
cp "$source" "/boot/loader/entries/$id.conf"
sed -i -e 's/^title .*/title LPIC initramfs rescue diagnostic/' -e "s@^initrd .*@initrd /initramfs-lpic-rescue-$k.img@" -e '/^options /s/$/ lpic.initramfs_rescue=1/' "/boot/loader/entries/$id.conf"
printf 'rescue-profile=storage-audit\n' > /etc/lpic-initramfs/payload
rm -f "/boot/initramfs-lpic-rescue-$k.img"
! test -e "/boot/initramfs-lpic-rescue-$k.img"
