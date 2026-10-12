#!/usr/bin/env bash
set -euo pipefail
test -d /run/systemd/system
install -d -m 0755 /var/lib/lpic-reboot /usr/local/libexec
cat /proc/sys/kernel/random/boot_id > /var/lib/lpic-reboot/initial-boot-id
printf 'archive-checkpoint=preserve-27\n' > /var/lib/lpic-reboot/checkpoint.txt
rm -f /var/lib/lpic-reboot/clean-exit
cat > /usr/local/libexec/lpic-reboot-worker <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
trap 'printf "worker=stopped-with-sigterm\n" > /var/lib/lpic-reboot/clean-exit; sync /var/lib/lpic-reboot/clean-exit; exit 0' TERM
while :; do
    sleep 1 &
    wait $! || true
done
EOF
chmod 0755 /usr/local/libexec/lpic-reboot-worker
cat > /etc/systemd/system/lpic-reboot-worker.service <<'EOF'
[Unit]
Description=LPIC archive worker during reboot
After=local-fs.target

[Service]
Type=simple
ExecStart=/usr/local/libexec/lpic-reboot-worker
KillSignal=SIGKILL
TimeoutStopSec=1
Restart=on-failure

[Install]
WantedBy=multi-user.target
EOF
systemctl daemon-reload
systemctl enable --now lpic-reboot-worker.service >/dev/null
systemctl is-active --quiet lpic-reboot-worker.service
test ! -e /var/lib/lpic-reboot/clean-exit
