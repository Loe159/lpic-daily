#!/usr/bin/env bash
set -euo pipefail
mkdir -p /opt/lpic-repo/pool /tmp/lpic-agent-pkg/DEBIAN /tmp/lpic-agent-pkg/usr/local/bin
cat > /tmp/lpic-agent-pkg/DEBIAN/control <<'EOF'
Package: lpic-alert-agent
Version: 1.0
Section: admin
Priority: optional
Architecture: all
Maintainer: LPIC Daily <noreply@example.invalid>
Description: Offline lab alert agent
Depends: bash
EOF
cat > /tmp/lpic-agent-pkg/usr/local/bin/lpic-alert-agent <<'EOF'
#!/bin/bash
printf 'alert-agent=ready\n'
EOF
chmod 0755 /tmp/lpic-agent-pkg/usr/local/bin/lpic-alert-agent
pkg=/opt/lpic-repo/pool/lpic-alert-agent_1.0_all.deb
dpkg-deb --build --root-owner-group /tmp/lpic-agent-pkg "$pkg" >/dev/null
dpkg-deb -f "$pkg" > /opt/lpic-repo/Packages
printf 'Filename: pool/lpic-alert-agent_1.0_all.deb\nSize: %s\nSHA256: %s\n\n' "$(wc -c < "$pkg" | tr -d ' ')" "$(sha256sum "$pkg" | cut -d' ' -f1)" >> /opt/lpic-repo/Packages
find /etc/apt/sources.list.d -maxdepth 1 -type f -delete
printf '\n' > /etc/apt/sources.list
printf 'deb [trusted=yes] file:/opt/lpic-broken-repo ./\n' > /etc/apt/sources.list.d/lpic-local.list
dpkg-query -W lpic-alert-agent >/dev/null 2>&1 && exit 1 || true
