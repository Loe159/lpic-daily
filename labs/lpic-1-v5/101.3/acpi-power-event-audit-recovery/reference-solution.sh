#!/usr/bin/env bash
set -euo pipefail
cat > /etc/acpi/events/lpic-power-audit <<'EOF'
event=button/power.*
action=/usr/bin/logger -t lpic-acpi-power lpic-guest-power-button
EOF
systemctl restart acpid.service
systemctl is-active --quiet acpid.service
# Dans le terminal parent LPIC Daily, saisir :acpi-power pour déclencher
# l'événement ACPI depuis libvirt avant d'exécuter :check.
