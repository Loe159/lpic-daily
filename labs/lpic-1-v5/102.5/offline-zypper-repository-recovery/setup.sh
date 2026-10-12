#!/usr/bin/env bash
set -euo pipefail
for command_name in rpmbuild createrepo_c zypper rpm; do
  command -v "$command_name" >/dev/null || { echo "Missing VM image package: $command_name" >&2; exit 1; }
done
mkdir -p /srv/lpic-zypp-repo /tmp/lpic-zypp-build/{BUILD,RPMS,SOURCES,SPECS,SRPMS} /tmp/lpic-zypp-src
cat > /tmp/lpic-zypp-src/runtime <<'EOF'
#!/bin/sh
printf 'zypp-agent=ready\n'
EOF
cat > /tmp/lpic-zypp-src/agent <<'EOF'
#!/bin/sh
set -eu
exec /usr/local/lib/lpic-archive-runtime
EOF
chmod 0755 /tmp/lpic-zypp-src/runtime /tmp/lpic-zypp-src/agent
cat > /tmp/lpic-zypp-build/SPECS/runtime.spec <<'EOF'
Name: lpic-archive-runtime
Version: 1.0
Release: 1
Summary: Offline LPIC archive runtime
License: MIT
BuildArch: noarch
%description
Runtime dependency for the LPIC recovery exercise.
%install
install -Dm0755 /tmp/lpic-zypp-src/runtime %{buildroot}/usr/local/lib/lpic-archive-runtime
%files
/usr/local/lib/lpic-archive-runtime
EOF
cat > /tmp/lpic-zypp-build/SPECS/agent.spec <<'EOF'
Name: lpic-archive-agent
Version: 1.0
Release: 1
Summary: Offline LPIC archive collector
License: MIT
BuildArch: noarch
Requires: lpic-archive-runtime >= 1.0
%description
Local archive collector requiring a packaged runtime.
%install
install -Dm0755 /tmp/lpic-zypp-src/agent %{buildroot}/usr/local/bin/lpic-archive-agent
%files
/usr/local/bin/lpic-archive-agent
EOF
rpmbuild --define '_topdir /tmp/lpic-zypp-build' -bb /tmp/lpic-zypp-build/SPECS/runtime.spec >/dev/null
rpmbuild --define '_topdir /tmp/lpic-zypp-build' -bb /tmp/lpic-zypp-build/SPECS/agent.spec >/dev/null
cp /tmp/lpic-zypp-build/RPMS/noarch/*.rpm /srv/lpic-zypp-repo/
createrepo_c /srv/lpic-zypp-repo >/dev/null
# Disable remote repositories inside this disposable VM. The task is intentionally offline.
zypper --non-interactive modifyrepo --disable --all >/dev/null
cat > /etc/zypp/repos.d/lpic-maint.repo <<'EOF'
[lpic-maint]
name=LPIC isolated recovery
enabled=1
autorefresh=0
baseurl=file:///srv/lpic-zypp-repo-missing
type=rpm-md
gpgcheck=0
EOF
! rpm -q lpic-archive-agent >/dev/null 2>&1
! rpm -q lpic-archive-runtime >/dev/null 2>&1
