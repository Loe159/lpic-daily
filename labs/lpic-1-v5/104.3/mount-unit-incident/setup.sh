#!/usr/bin/env bash
set -euo pipefail
test -b /dev/vdb
mkfs.ext4 -F -q -L lpicreport /dev/vdb
mkdir -p /srv/lpic/reports
mount /dev/vdb /srv/lpic/reports
printf 'report-batch=2026-10-09\n' > /srv/lpic/reports/manifest.txt
umount /srv/lpic/reports
cat > /etc/systemd/system/srv-lpic-reports.mount <<'EOF'
[Unit]
Description=LPIC Reports Disk
[Mount]
What=/dev/disk/by-label/wrong-report
Where=/srv/lpic/reports
Type=ext4
Options=nodev,nosuid
TimeoutSec=5s
[Install]
WantedBy=multi-user.target
EOF
systemctl daemon-reload
systemctl enable srv-lpic-reports.mount
