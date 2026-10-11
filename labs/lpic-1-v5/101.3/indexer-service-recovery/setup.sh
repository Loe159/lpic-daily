#!/usr/bin/env bash
set -euo pipefail
test -d /run/systemd/system
install -d -m 0755 /var/lib/lpic-indexer /usr/local/libexec
printf 'ERROR ssh user22 rejected\nINFO collector started\n' > /var/lib/lpic-indexer/events.log
cat > /usr/local/libexec/lpic-indexer <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
while :; do
  awk '/^ERROR / {print}' /var/lib/lpic-indexer/events.log > /var/lib/lpic-indexer/index.next
  mv -f /var/lib/lpic-indexer/index.next /var/lib/lpic-indexer/index.log
  sleep 0.2
done
EOF
chmod 0755 /usr/local/libexec/lpic-indexer
rm -f /var/lib/lpic-indexer/index.log /var/lib/lpic-indexer/index.next
cat > /etc/systemd/system/lpic-indexer.service <<'EOF'
[Unit]
Description=LPIC event indexer
After=network.target
Wants=network.target

[Service]
Type=simple
ExecStart=/usr/bin/lpic-missing-agent
Restart=no

[Install]
WantedBy=multi-user.target
EOF
systemctl daemon-reload
systemctl enable lpic-indexer.service
systemctl start lpic-indexer.service >/dev/null 2>&1 || true
! systemctl is-active --quiet lpic-indexer.service
