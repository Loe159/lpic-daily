#!/usr/bin/env bash
set -euo pipefail
# Guest-only, never run on a workstation or outside LPIC-Daily libvirt.
command -v efibootmgr >/dev/null
test "$(systemd-detect-virt --vm)" = kvm
test -b /dev/vda
test -d /sys/firmware/efi/efivars
state=/var/lib/lpic-daily/uefi-boot-order
test ! -e "$state/target-order"
initial="$(efibootmgr)"
if printf '%s\n' "$initial" | grep -Eq '^Boot[[:xdigit:]]{4}\*?[[:space:]]+LPIC-(primary|rescue)([[:space:]]|$)'; then
  echo 'Entrées LPIC préexistantes : réinitialiser la VM' >&2
  exit 1
fi
if printf '%s\n' "$initial" | grep -q '^BootNext:'; then
  echo 'BootNext préexistant : configuration inattendue' >&2
  exit 1
fi
original="$(printf '%s\n' "$initial" | sed -nE 's/^BootOrder:[[:space:]]*([[:xdigit:],]+).*$/\1/p')"
[[ "$original" =~ ^[[:xdigit:]]{4}(,[[:xdigit:]]{4})*$ ]]
mountpoint -q /boot/efi || mount /boot/efi
test "$(findmnt -n -o FSTYPE --mountpoint /boot/efi)" = vfat
source_efi=''
for candidate in /boot/efi/EFI/fedora/shimx64.efi /boot/efi/EFI/fedora/grubx64.efi /boot/efi/EFI/BOOT/BOOTX64.EFI; do
  if test -s "$candidate"; then source_efi="$candidate"; break; fi
done
test -n "$source_efi"
file -b "$source_efi" | grep -Eq '^PE32'
esp_device="$(readlink -f "$(findmnt -n -o SOURCE --mountpoint /boot/efi)")"
test -b "$esp_device"
part_no="$(lsblk -no PARTN "$esp_device" | tr -d '[:space:]')"
disk_name="$(lsblk -no PKNAME "$esp_device" | tr -d '[:space:]')"
[[ "$part_no" =~ ^[0-9]+$ ]]
[[ "$disk_name" =~ ^[a-zA-Z0-9]+$ ]]
test -b "/dev/$disk_name"
install -d -m 0700 "$state"
for label in primary rescue; do
  install -d -m 0755 "/boot/efi/EFI/LPIC-$label"
  cp -a "$(dirname "$source_efi")/." "/boot/efi/EFI/LPIC-$label/"
done
loader_name="$(basename "$source_efi")"
sha256sum "/boot/efi/EFI/LPIC-primary/$loader_name" "/boot/efi/EFI/LPIC-rescue/$loader_name" > "$state/loaders.sha256"
efibootmgr -c -d "/dev/$disk_name" -p "$part_no" -L LPIC-primary -l "\\EFI\\LPIC-primary\\$loader_name"
efibootmgr -c -d "/dev/$disk_name" -p "$part_no" -L LPIC-rescue -l "\\EFI\\LPIC-rescue\\$loader_name"
boot_id() {
  local label="$1"
  efibootmgr | sed -nE "s/^Boot([[:xdigit:]]{4})\*?[[:space:]]+$label[[:space:]]*$/\1/p" | tr '[:lower:]' '[:upper:]'
}
primary="$(boot_id LPIC-primary)"
rescue="$(boot_id LPIC-rescue)"
[[ "$primary" =~ ^[[:xdigit:]]{4}$ ]]
[[ "$rescue" =~ ^[[:xdigit:]]{4}$ ]]
test "$primary" != "$rescue"
# OVMF may auto-discover an additional Fedora entry while efibootmgr creates
# the LPIC options. Preserve the current non-LPIC order instead of comparing
# against a stale snapshot taken before firmware enumeration.
current_order="$(efibootmgr | sed -nE 's/^BootOrder:[[:space:]]*([[:xdigit:],]+).*$/\\1/p')"
[[ "$current_order" =~ ^[[:xdigit:]]{4}(,[[:xdigit:]]{4})*$ ]]
original=''
IFS=',' read -ra entries <<< "$current_order"
for entry in "${entries[@]}"; do
  entry="${entry^^}"
  if [[ "$entry" == "$primary" || "$entry" == "$rescue" ]]; then
    continue
  fi
  if [[ -n "$original" ]]; then original+=","; fi
  original+="$entry"
done
[[ -n "$original" ]]
target="$primary,$rescue,$original"
broken="$rescue,$primary,$original"
printf '%s\n' "$original" > "$state/original-order"
printf '%s\n' "$target" > "$state/target-order"
efibootmgr -o "$broken"
test "$(efibootmgr | sed -nE 's/^BootOrder:[[:space:]]*([[:xdigit:],]+).*$/\1/p')" = "$broken"
