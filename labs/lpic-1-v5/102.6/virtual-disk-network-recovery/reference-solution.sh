#!/usr/bin/env bash
set -euo pipefail
iface=$(cat /var/lib/lpic-vm-resources/iface)
nmcli --wait 30 device connect "$iface"
mount /srv/lpic-virtual-journal
systemctl reset-failed lpic-virtual-journal.service || true
systemctl start lpic-virtual-journal.service
