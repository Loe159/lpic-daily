#!/usr/bin/env bash
set -euo pipefail
chown root:project /srv/shared
chmod 3770 /srv/shared
chmod 0755 /srv/shared/audit-helper
cat > /root/.bashrc <<'EOF'
umask 0007
EOF
rm -f /srv/shared/team-note
/usr/bin/bash --noprofile --rcfile /root/.bashrc -ic ': > /srv/shared/team-note'
