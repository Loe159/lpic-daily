#!/usr/bin/env bash
set -euo pipefail
test -d /run/systemd/system
install -d -m 0755 /var/lib/lpic-daily/boot-journal /etc/systemd/journald.conf.d
cat /proc/sys/kernel/random/boot_id > /var/lib/lpic-daily/boot-journal/initial-boot-id
cat > /etc/systemd/journald.conf.d/95-lpic-incident.conf <<'EOF'
[Journal]
Storage=volatile
EOF
systemctl restart systemd-journald.service
logger -t lpic-bootdiag -- 'lpic-incident=journal-recovery'
journalctl --boot=0 --no-pager --output=cat -t lpic-bootdiag | grep -Fxq 'lpic-incident=journal-recovery'
