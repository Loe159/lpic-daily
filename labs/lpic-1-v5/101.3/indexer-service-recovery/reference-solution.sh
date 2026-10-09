#!/usr/bin/env bash
set -euo pipefail
sed -i 's|ExecStart=/usr/bin/lpic-missing-agent|ExecStart=/usr/bin/sleep infinity|' /etc/systemd/system/lpic-indexer.service
systemctl daemon-reload
systemctl restart lpic-indexer.service
systemctl is-active --quiet lpic-indexer.service
