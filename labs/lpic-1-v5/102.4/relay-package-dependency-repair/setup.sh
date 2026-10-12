#!/usr/bin/env bash
set -euo pipefail
mkdir -p /opt/lpic-debs /tmp/lpic-relay-runtime/DEBIAN /tmp/lpic-relay-runtime/usr/local/lib /tmp/lpic-relay-agent/DEBIAN /tmp/lpic-relay-agent/usr/local/bin
cat > /tmp/lpic-relay-runtime/DEBIAN/control <<'EOF'
Package: lpic-relay-runtime
Version: 1.0
Section: admin
Priority: optional
Architecture: all
Maintainer: LPIC Daily <noreply@example.invalid>
Description: Local relay runtime
EOF
cat > /tmp/lpic-relay-agent/DEBIAN/control <<'EOF'
Package: lpic-relay-agent
Version: 1.0
Section: admin
Priority: optional
Architecture: all
Maintainer: LPIC Daily <noreply@example.invalid>
Depends: lpic-relay-runtime (>= 1.0)
Description: Offline relay agent
EOF
printf '#!/bin/sh\nprintf "relay=ready\\n"\n' > /tmp/lpic-relay-runtime/usr/local/lib/lpic-relay-runtime
printf '#!/bin/sh\nset -eu\nexec /usr/local/lib/lpic-relay-runtime\n' > /tmp/lpic-relay-agent/usr/local/bin/lpic-relay-agent
chmod 0755 /tmp/lpic-relay-runtime/usr/local/lib/lpic-relay-runtime /tmp/lpic-relay-agent/usr/local/bin/lpic-relay-agent
dpkg-deb --build --root-owner-group /tmp/lpic-relay-runtime /opt/lpic-debs/lpic-relay-runtime_1.0_all.deb >/dev/null
dpkg-deb --build --root-owner-group /tmp/lpic-relay-agent /opt/lpic-debs/lpic-relay-agent_1.0_all.deb >/dev/null
if dpkg -i /opt/lpic-debs/lpic-relay-agent_1.0_all.deb >/dev/null 2>&1; then
  echo 'Expected missing dependency did not occur' >&2; exit 1
fi
test "$(dpkg-query -W -f='${Status}' lpic-relay-agent)" = 'install ok unpacked'
