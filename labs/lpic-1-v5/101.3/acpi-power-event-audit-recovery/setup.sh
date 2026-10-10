#!/usr/bin/env bash
set -euo pipefail
test -d /run/systemd/system
command -v logger >/dev/null
command -v acpid >/dev/null
systemctl is-active --quiet qemu-guest-agent.service
install -d -m 0755 /var/lib/lpic-acpi-audit /etc/acpi/events /etc/systemd/logind.conf.d
cat /proc/sys/kernel/random/boot_id > /var/lib/lpic-acpi-audit/boot-id

# Never let a learner-triggered ACPI event shut down the disposable VM.
cat > /etc/systemd/logind.conf.d/95-lpic-acpi-lab.conf <<'EOF'
[Login]
HandlePowerKey=ignore
HandlePowerKeyLongPress=ignore
EOF
systemctl restart systemd-logind.service
systemctl is-active --quiet qemu-guest-agent.service

# Isolate this disposable VM's ACPI event rules. A distribution-supplied
# power-button handler must not shut down the lab independently of logind.
install -d -m 0700 /var/lib/lpic-acpi-audit/previous-events
for rule in /etc/acpi/events/*; do
  test -f "$rule" || continue
  mv "$rule" /var/lib/lpic-acpi-audit/previous-events/
done

# Broken audit handler: ACPI event is caught, but never journaled.
cat > /etc/acpi/events/lpic-power-audit <<'EOF'
event=button/power.*
action=/usr/bin/true
EOF
systemctl enable --now acpid.service
systemctl restart acpid.service
systemctl is-active --quiet acpid.service
date +%s > /var/lib/lpic-acpi-audit/start-epoch
! journalctl -b --since "@$(cat /var/lib/lpic-acpi-audit/start-epoch)" -t lpic-acpi-power --no-pager -o cat | grep -Fxq lpic-guest-power-button
