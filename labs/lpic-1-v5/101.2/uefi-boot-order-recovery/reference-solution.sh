#!/usr/bin/env bash
set -euo pipefail

# Operate only inside a disposable libvirt/OVMF guest.
trap 'rc=$?; printf "UEFI reference failed (line %s, exit %s): %s\n" "$LINENO" "$rc" "$BASH_COMMAND" >&2' ERR
test "$(systemd-detect-virt --vm)" = kvm
test -b /dev/vda
test -d /sys/firmware/efi/efivars

# The setup writes the expected full BootOrder while the instance is still
# inaccessible to the learner. Its immutable copy is separately held by
# the libvirt host-side checker; this file is only an input to the reference
# repair, never authoritative grading evidence.
expected="$(cat /var/lib/lpic-daily/uefi-boot-order/target-order)"
expected="${expected^^}"
[[ "$expected" =~ ^[[:xdigit:]]{4}(,[[:xdigit:]]{4})+$ ]]

# Demonstrate and confirm the actual firmware recovery (not a text-only edit).
before="$(efibootmgr)"
order="$(printf '%s\n' "$before" | sed -nE 's/^BootOrder:[[:space:]]*([[:xdigit:],]+).*$/\1/p')"
[[ -n "$order" ]]
[[ "${order^^}" != "$expected" ]]

# A temporary BootNext override must not take precedence over BootOrder.
if printf '%s\n' "$before" | grep -q '^BootNext:'; then
  efibootmgr -N
fi
efibootmgr -o "$expected"
after="$(efibootmgr)"
restored="$(printf '%s\n' "$after" | sed -nE 's/^BootOrder:[[:space:]]*([[:xdigit:],]+).*$/\1/p')"
test "${restored^^}" = "$expected"
! printf '%s\n' "$after" | grep -q '^BootNext:'
