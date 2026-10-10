#!/usr/bin/env bash
set -euo pipefail
test -d /run/systemd/system
install -d -m 0755 /var/lib/lpic-maintenance /usr/local/libexec
printf 'archive-batch=retain-17\n' > /var/lib/lpic-maintenance/archive-batch.txt
cat /proc/sys/kernel/random/boot_id > /var/lib/lpic-maintenance/initial-boot-id
cat > /usr/local/libexec/lpic-archive-worker <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
trap 'printf "worker=stopped-gracefully\n" > /var/lib/lpic-maintenance/stop-event; exit 0' TERM
while :; do
  sleep 1 &
  wait $! || true
done
EOF
chmod 0755 /usr/local/libexec/lpic-archive-worker
cat > /etc/systemd/system/lpic-archive-worker.service <<'EOF'
[Unit]
Description=LPIC archiving worker
[Service]
Type=simple
ExecStart=/usr/local/libexec/lpic-archive-worker
KillSignal=SIGTERM
TimeoutStopSec=15
Restart=no
EOF
cat > /etc/systemd/system/lpic-maintenance.target <<'EOF'
[Unit]
Description=LPIC application maintenance target
Conflicts=lpic-archive-worker.service
After=lpic-archive-worker.service
EOF
systemctl daemon-reload
systemctl start lpic-archive-worker.service
systemctl is-active --quiet lpic-archive-worker.service
test ! -e /var/lib/lpic-maintenance/stop-event
test ! -e /var/lib/lpic-maintenance/forced-kill
