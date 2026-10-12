#!/usr/bin/env bash
set -euo pipefail
test -d /run/systemd/system && test -b /dev/vdb
command -v lspci >/dev/null && command -v mkfs.ext4 >/dev/null && command -v debugfs >/dev/null
mkdir -p /srv/lpic-vault
mkfs.ext4 -F -q -L LPICVAULT /dev/vdb
printf 'vault-index=survive-41\n' > /tmp/lpic-vault-index.txt
debugfs -w -R 'write /tmp/lpic-vault-index.txt /catalog.txt' /dev/vdb >/dev/null 2>&1
cat >> /etc/fstab <<'EOF'
LABEL=LPICVAULT /srv/lpic-vault ext4 defaults,nofail 0 2
EOF
cat > /etc/systemd/system/lpic-vault-reader.service <<'EOF'
[Unit]
Description=LPIC archive index reader
After=local-fs.target

[Service]
Type=oneshot
ExecStart=/usr/bin/grep -Fxq vault-index=survive-41 /srv/lpic-vault/catalog.txt
RemainAfterExit=yes

[Install]
WantedBy=multi-user.target
EOF
systemctl daemon-reload
systemctl enable lpic-vault-reader.service >/dev/null
block=$(readlink -f /sys/block/vdb/device)
pci_bdf=''
for pci in /sys/bus/pci/devices/*; do
  pci_path=$(readlink -f "$pci") || continue
  if [[ "$block" == "$pci_path/"* ]]; then
    pci_bdf=$(basename "$pci")
    break
  fi
done
test -n "$pci_bdf"
test -L "/sys/bus/pci/devices/$pci_bdf/driver"
test "$(basename "$(readlink -f "/sys/bus/pci/devices/$pci_bdf/driver")")" = virtio-pci
echo "$pci_bdf" > /sys/bus/pci/drivers/virtio-pci/unbind
udevadm settle
logger -t hypervisor-event "Volume d'archives: le contrôleur PCI $pci_bdf n'est plus associé au pilote virtio"
test ! -b /dev/vdb
systemctl start lpic-vault-reader.service >/dev/null 2>&1 || true
