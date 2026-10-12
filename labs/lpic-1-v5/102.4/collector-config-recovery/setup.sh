#!/usr/bin/env bash
set -euo pipefail
command -v dpkg-reconfigure >/dev/null
mkdir -p /opt/lpic-debs /tmp/lpic-collector/DEBIAN /tmp/lpic-collector/usr/local/bin /var/lib/lpic-collector
cat > /tmp/lpic-collector/DEBIAN/control <<'EOF'
Package: lpic-collector-agent
Version: 1.0
Section: admin
Priority: optional
Architecture: all
Maintainer: LPIC Daily <noreply@example.invalid>
Depends: bash
Description: Local configurable collector agent
EOF
cat > /tmp/lpic-collector/DEBIAN/postinst <<'EOF'
#!/bin/sh
set -e
if [ "$1" = configure ]; then
  install -d -m 0755 /etc/lpic-collector
  printf 'mode=collect\nendpoint=unix-local\n' > /etc/lpic-collector/agent.conf
fi
EOF
chmod 0755 /tmp/lpic-collector/DEBIAN/postinst
cat > /tmp/lpic-collector/usr/local/bin/lpic-collector-agent <<'EOF'
#!/bin/sh
set -eu
grep -Fxq 'mode=collect' /etc/lpic-collector/agent.conf
grep -Fxq 'endpoint=unix-local' /etc/lpic-collector/agent.conf
printf 'collector=ready\n'
EOF
chmod 0755 /tmp/lpic-collector/usr/local/bin/lpic-collector-agent
dpkg-deb --build --root-owner-group /tmp/lpic-collector /opt/lpic-debs/lpic-collector-agent_1.0_all.deb >/dev/null
dpkg -i /opt/lpic-debs/lpic-collector-agent_1.0_all.deb >/dev/null
printf 'mode=disabled\nendpoint=missing\n' > /etc/lpic-collector/agent.conf
printf 'retain=customer-batch-991\n' > /var/lib/lpic-collector/batch.txt
test "$(dpkg-query -W -f='${Status}' lpic-collector-agent)" = 'install ok installed'
if /usr/local/bin/lpic-collector-agent >/dev/null 2>&1; then
  echo 'Setup error: agent should be broken' >&2
  exit 1
fi
