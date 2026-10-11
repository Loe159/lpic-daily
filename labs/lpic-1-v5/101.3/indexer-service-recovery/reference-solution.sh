#!/usr/bin/env bash
set -euo pipefail
sed -i 's|^ExecStart=/usr/bin/lpic-missing-agent$|ExecStart=/usr/local/libexec/lpic-indexer|' /etc/systemd/system/lpic-indexer.service
systemctl daemon-reload
systemctl reset-failed lpic-indexer.service || true
systemctl start lpic-indexer.service
systemctl is-active --quiet lpic-indexer.service
for attempt in $(seq 1 30); do
  if grep -Fxq 'ERROR ssh user22 rejected' /var/lib/lpic-indexer/index.log 2>/dev/null; then
    exit 0
  fi
  sleep 0.2
done
echo "Indexer did not process the initial event" >&2
exit 1
