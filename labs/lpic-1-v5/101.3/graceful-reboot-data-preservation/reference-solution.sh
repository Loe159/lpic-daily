#!/usr/bin/env bash
set -euo pipefail
mkdir -p /etc/systemd/system/lpic-reboot-worker.service.d
cat > /etc/systemd/system/lpic-reboot-worker.service.d/20-graceful-stop.conf <<'EOF'
[Service]
KillSignal=SIGTERM
TimeoutStopSec=20
EOF
systemctl daemon-reload
# Le redémarrage doit être demandé dans le terminal parent avec :reboot.
