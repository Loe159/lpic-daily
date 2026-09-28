#!/usr/bin/env bash
set -euo pipefail
chown root:project /srv/project-transfer/incoming /srv/project-transfer/team
chmod 1770 /srv/project-transfer/incoming
chmod 2770 /srv/project-transfer/team
chmod 0755 /srv/project-transfer/audit-helper
cat > /root/.bashrc <<'EOF'
umask 0007
EOF
rm -f /srv/project-transfer/team/handoff-note
/usr/bin/bash --noprofile --rcfile /root/.bashrc -ic ': > /srv/project-transfer/team/handoff-note'
