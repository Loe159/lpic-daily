#!/usr/bin/env bash
set -euo pipefail
command -v rpmbuild >/dev/null && command -v createrepo_c >/dev/null && command -v dnf >/dev/null
mkdir -p /srv/lpic-rpm-repo /tmp/lpic-rpm-build/{BUILD,RPMS,SOURCES,SPECS,SRPMS} /tmp/lpic-rpm-src
cat > /tmp/lpic-rpm-src/runtime <<'EOF'
#!/bin/sh
printf 'dnf-agent=ready\n'
EOF
cat > /tmp/lpic-rpm-src/agent <<'EOF'
#!/bin/sh
set -eu
exec /usr/local/lib/lpic-dnf-runtime
EOF
chmod +x /tmp/lpic-rpm-src/runtime /tmp/lpic-rpm-src/agent
cat > /tmp/lpic-rpm-build/SPECS/runtime.spec <<'EOF'
Name: lpic-dnf-runtime
Version: 1.0
Release: 1
Summary: LPIC offline runtime
License: MIT
BuildArch: noarch
%description
Runtime dependency for the LPIC offline diagnostic agent.
%install
install -Dm0755 /tmp/lpic-rpm-src/runtime %{buildroot}/usr/local/lib/lpic-dnf-runtime
%files
/usr/local/lib/lpic-dnf-runtime
EOF
cat > /tmp/lpic-rpm-build/SPECS/agent.spec <<'EOF'
Name: lpic-dnf-agent
Version: 1.0
Release: 1
Summary: LPIC offline diagnostic agent
License: MIT
BuildArch: noarch
Requires: lpic-dnf-runtime >= 1.0
%description
Offline diagnostic agent with a runtime dependency.
%install
install -Dm0755 /tmp/lpic-rpm-src/agent %{buildroot}/usr/local/bin/lpic-dnf-agent
%files
/usr/local/bin/lpic-dnf-agent
EOF
rpmbuild --define '_topdir /tmp/lpic-rpm-build' -bb /tmp/lpic-rpm-build/SPECS/runtime.spec >/dev/null
rpmbuild --define '_topdir /tmp/lpic-rpm-build' -bb /tmp/lpic-rpm-build/SPECS/agent.spec >/dev/null
cp /tmp/lpic-rpm-build/RPMS/noarch/*.rpm /srv/lpic-rpm-repo/
createrepo_c /srv/lpic-rpm-repo >/dev/null
cat > /etc/yum.repos.d/lpic-offline.repo <<'EOF'
[lpic-offline]
name=LPIC isolated maintenance repository
baseurl=file:///srv/lpic-rpm-repo-old
enabled=1
gpgcheck=0
EOF
if rpm -q lpic-dnf-agent >/dev/null 2>&1; then echo 'agent unexpectedly installed' >&2; exit 1; fi
