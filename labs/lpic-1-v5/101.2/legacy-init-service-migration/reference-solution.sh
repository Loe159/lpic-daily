#!/usr/bin/env bash
set -euo pipefail
sed -i 's|^ExecStart=/etc/init/lpic-legacy-indexer.conf$|ExecStart=/usr/local/libexec/lpic-legacy-indexer|' /etc/systemd/system/lpic-legacy-indexer.service
systemctl daemon-reload
systemctl reset-failed lpic-legacy-indexer.service
systemctl enable --now lpic-legacy-indexer.service
systemctl is-active --quiet lpic-legacy-indexer.service
