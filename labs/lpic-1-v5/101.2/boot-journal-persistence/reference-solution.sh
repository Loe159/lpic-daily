#!/usr/bin/env bash
set -euo pipefail
rm -f /etc/systemd/journald.conf.d/95-lpic-incident.conf
install -d -m 2755 /var/log/journal
cat > /etc/systemd/journald.conf.d/90-lpic-persistent.conf <<'EOF'
[Journal]
Storage=persistent
EOF
systemctl restart systemd-journald.service
journalctl --flush
journalctl --boot=0 --no-pager --output=cat -t lpic-bootdiag | grep -Fxq 'lpic-incident=journal-recovery'
# Use the lab's :reboot command; do not reboot within the setup/reference runner.
