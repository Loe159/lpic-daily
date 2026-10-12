#!/usr/bin/env bash
set -euo pipefail
# Operate only on disposable libvirt/OVMF guests.
test "$(systemd-detect-virt --vm)" = kvm
test -b /dev/vda
test -d /sys/firmware/efi/efivars
state="$(efibootmgr)"
boot_id() {
  printf '%s\n' "$state" | sed -nE "s/^Boot([[:xdigit:]]{4})\\*?[[:space:]]+LPIC-$1([[:space:]]|$).*$/\\1/p" | tr '[:lower:]' '[:upper:]'
}
primary="$(boot_id primary)"
rescue="$(boot_id rescue)"
[[ "$primary" =~ ^[[:xdigit:]]{4}$ ]]
[[ "$rescue" =~ ^[[:xdigit:]]{4}$ ]]
test "$primary" != "$rescue"
order="$(printf '%s\n' "$state" | sed -nE 's/^BootOrder:[[:space:]]*([[:xdigit:],]+).*$/\\1/p' | tr '[:lower:]' '[:upper:]')"
[[ "$order" =~ ^[[:xdigit:]]{4}(,[[:xdigit:]]{4})*$ ]]
other=()
IFS=',' read -ra entries <<< "$order"
for entry in "${entries[@]}"; do
  entry="${entry^^}"
  if [[ "$entry" != "$primary" && "$entry" != "$rescue" ]]; then
    other+=("$entry")
  fi
done
test "${#other[@]}" -gt 0
target="$primary,$rescue"
for entry in "${other[@]}"; do target+=",$entry"; done
if printf '%s\n' "$state" | grep -q '^BootNext:'; then efibootmgr -N; fi
efibootmgr -o "$target"
