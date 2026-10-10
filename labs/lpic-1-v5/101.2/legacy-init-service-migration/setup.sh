#!/usr/bin/env bash
set -euo pipefail
test "$(ps -p 1 -o comm= | xargs)" = systemd
command -v systemctl >/dev/null
install -d -m 0755 /var/lib/lpic-init-migration /etc/init /etc/init.d /usr/local/libexec

# This is a real running service, not a report file submitted as proof.
cat > /usr/local/libexec/lpic-legacy-indexer <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
install -d -m 0755 /run/lpic-legacy-indexer
while :; do
  date +%s%N > /run/lpic-legacy-indexer/heartbeat
  sleep 1
done
EOF
chmod 0755 /usr/local/libexec/lpic-legacy-indexer

# Historical init artifacts for diagnosis only. Fedora PID 1 remains systemd.
cat > /etc/init.d/lpic-legacy-indexer <<'EOF'
#!/bin/sh
### BEGIN INIT INFO
# Provides: lpic-legacy-indexer
# Required-Start: $local_fs
# Required-Stop: $local_fs
# Default-Start: 2 3 4 5
# Default-Stop: 0 1 6
# Short-Description: Legacy archive indexer
### END INIT INFO
case "$1" in
 start) /usr/local/libexec/lpic-legacy-indexer & ;;
 stop) pkill -f '^/usr/local/libexec/lpic-legacy-indexer$' || : ;;
 status) pgrep -f '^/usr/local/libexec/lpic-legacy-indexer$' >/dev/null ;;
 *) exit 2 ;;
esac
EOF
chmod 0755 /etc/init.d/lpic-legacy-indexer
cat > /etc/init/lpic-legacy-indexer.conf <<'EOF'
description "Legacy archive indexer (historical Upstart job)"
start on runlevel [2345]
stop on runlevel [016]
respawn
exec /usr/local/libexec/lpic-legacy-indexer
EOF
( cd / && sha256sum etc/init.d/lpic-legacy-indexer etc/init/lpic-legacy-indexer.conf ) > /var/lib/lpic-init-migration/archive-sha256

# The migration bug: ExecStart points at a declarative Upstart .conf.
cat > /etc/systemd/system/lpic-legacy-indexer.service <<'EOF'
[Unit]
Description=LPIC migrated legacy indexer
After=local-fs.target

[Service]
Type=simple
ExecStart=/etc/init/lpic-legacy-indexer.conf
Restart=on-failure
RestartSec=1s

[Install]
WantedBy=multi-user.target
EOF
systemctl daemon-reload
systemctl enable lpic-legacy-indexer.service
systemctl start lpic-legacy-indexer.service >/dev/null 2>&1 || :
! systemctl is-active --quiet lpic-legacy-indexer.service
test ! -e /run/lpic-legacy-indexer/heartbeat
