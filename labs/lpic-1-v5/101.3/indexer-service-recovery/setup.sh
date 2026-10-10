#!/usr/bin/env bash
set -euo pipefail
test -d /run/systemd/system
cat > /etc/systemd/system/lpic-indexer.service <<'EOF'
[Unit]
Description=LPIC indexer practice service
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
