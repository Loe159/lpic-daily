#!/usr/bin/env bash
set -euo pipefail
mkdir -p /root/lpic-state /var/lib/lpic
systemctl set-default multi-user.target
{
  readlink -f /usr/lib/systemd/system/runlevel3.target
  readlink -f /usr/lib/systemd/system/runlevel1.target
} > /root/lpic-state/runlevel-map.txt
systemctl cat rescue.target > /root/lpic-state/rescue.txt
cat > /etc/systemd/system/lpic-base.service <<'EOF'
[Unit]
Description=LPIC base
[Service]
Type=oneshot
ExecStart=/usr/bin/true
RemainAfterExit=yes
EOF
cat > /etc/systemd/system/lpic-dependent.service <<'EOF'
[Unit]
Description=LPIC dependent
Requires=lpic-base.service
After=lpic-base.service
[Service]
Type=oneshot
ExecStart=/usr/bin/true
RemainAfterExit=yes
[Install]
WantedBy=multi-user.target
EOF
cat > /etc/systemd/system/lpic-cleanstop.service <<'EOF'
[Unit]
Description=LPIC clean stop evidence
[Service]
Type=simple
ExecStart=/usr/bin/sleep infinity
ExecStop=/usr/bin/sh -c 'printf "clean-stop\n" > /var/lib/lpic/clean-stop'
[Install]
WantedBy=multi-user.target
EOF
systemctl daemon-reload
systemctl enable --now lpic-dependent.service lpic-cleanstop.service
printf 'LPIC maintenance\n' | wall || true
logger -t lpic-wall 'LPIC maintenance'
systemctl cat acpid.service > /root/lpic-state/acpi.txt
printf 'Configuration prête. Utilise :reboot puis :check.\n'
