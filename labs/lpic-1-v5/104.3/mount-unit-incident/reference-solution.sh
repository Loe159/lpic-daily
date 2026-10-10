#!/usr/bin/env bash
set -euo pipefail
sed -i 's|by-label/wrong-report|by-label/lpicreport|' /etc/systemd/system/srv-lpic-reports.mount
systemctl daemon-reload
systemctl start srv-lpic-reports.mount
