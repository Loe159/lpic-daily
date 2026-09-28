#!/usr/bin/env bash
set -euo pipefail
chown root:project /srv/project-transfer
chmod 3770 /srv/project-transfer
chmod 0755 /srv/project-transfer/audit-helper
cat > /root/.bashrc <<'EOF'
umask 0007
EOF
rm -f /srv/project-transfer/handoff-note
/usr/bin/bash --noprofile --rcfile /root/.bashrc -ic ': > /srv/project-transfer/handoff-note'
