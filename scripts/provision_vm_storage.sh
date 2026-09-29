#!/usr/bin/env bash
set -euo pipefail

if [[ "${EUID}" -ne 0 ]]; then
  echo "Run this provisioning step as root, for example: sudo scripts/provision_vm_storage.sh" >&2
  exit 1
fi

target_user="${SUDO_USER:-}"
if [[ -z "${target_user}" || "${target_user}" == "root" ]]; then
  echo "SUDO_USER must identify the regular LPIC Daily user." >&2
  exit 1
fi

target_uid="$(id -u "${target_user}")"
target_group="$(id -gn "${target_user}")"
root="/var/lib/libvirt/images/lpic-daily/${target_uid}"

install -d -m 0755 /var/lib/libvirt/images/lpic-daily
install -d -m 0755 -o "${target_user}" -g "${target_group}" "${root}"
install -d -m 0755 -o "${target_user}" -g "${target_group}" "${root}/images" "${root}/state"

if command -v restorecon >/dev/null 2>&1; then
  restorecon -RF "${root}" || {
    echo "SELinux relabel failed for ${root}" >&2
    exit 1
  }
fi

echo "LPIC Daily VM storage provisioned for ${target_user}:"
echo "  images: ${root}/images"
echo "  state:  ${root}/state"
