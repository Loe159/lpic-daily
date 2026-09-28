#!/usr/bin/env bash
set -euo pipefail
getent group project >/dev/null
id alice >/dev/null 2>&1
id bob >/dev/null 2>&1
install -d -o root -g root -m 0755 /srv/shared
rm -rf /srv/shared/*
cat > /root/.bashrc <<'EOF'
umask 0022
EOF
cp /usr/bin/true /srv/shared/audit-helper
chown root:root /srv/shared/audit-helper
chmod 4755 /srv/shared/audit-helper
