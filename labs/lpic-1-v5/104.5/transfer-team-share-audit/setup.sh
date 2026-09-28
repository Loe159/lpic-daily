#!/usr/bin/env bash
set -euo pipefail
install -d -o root -g root -m 0777 /srv/project-transfer
rm -rf /srv/project-transfer/*
install -d -o root -g root -m 0777 /srv/project-transfer/incoming
install -d -o root -g root -m 0777 /srv/project-transfer/team
cat > /root/.bashrc <<'EOF'
umask 0022
EOF
cp /usr/bin/true /srv/project-transfer/audit-helper
chown root:root /srv/project-transfer/audit-helper
chmod 4755 /srv/project-transfer/audit-helper
