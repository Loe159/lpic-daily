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
global_network_lock="/var/lib/libvirt/images/lpic-daily/.network-allocation.lock"
if [[ ! -e "${global_network_lock}" ]]; then
  install -m 0666 -o root -g root /dev/null "${global_network_lock}"
else
  if [[ ! -f "${global_network_lock}" || -L "${global_network_lock}" ]]; then
    echo "Refusing unsafe global network lock path: ${global_network_lock}" >&2
    exit 1
  fi
  chown root:root "${global_network_lock}"
  chmod 0666 "${global_network_lock}"
fi
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
echo "  network lock: ${global_network_lock}"
