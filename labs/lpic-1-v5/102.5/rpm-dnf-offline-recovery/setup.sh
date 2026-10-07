#!/usr/bin/env bash
set -euo pipefail
rm -rf /workspace/*
mkdir -p /workspace/rpmbuild/{BUILD,BUILDROOT,RPMS,SOURCES,SPECS,SRPMS} /workspace/repo
cat > /workspace/rpmbuild/SPECS/lpic-helper.spec <<'EOF'
Name: lpic-helper
Version: 1.0
Release: 1
Summary: LPIC Daily helper fixture
License: MIT
BuildArch: noarch
%description
Offline dependency fixture.
%install
mkdir -p %{buildroot}/usr/share/lpic-helper
printf 'helper-ok\n' > %{buildroot}/usr/share/lpic-helper/state.txt
%files
/usr/share/lpic-helper/state.txt
EOF
cat > /workspace/rpmbuild/SPECS/lpic-demo.spec <<'EOF'
Name: lpic-demo
Version: 1.0
Release: 1
Summary: LPIC Daily RPM fixture
License: MIT
BuildArch: noarch
Requires: lpic-helper >= 1.0
%description
Offline RPM practice fixture.
%install
mkdir -p %{buildroot}/usr/bin
cat > %{buildroot}/usr/bin/lpic-demo <<'SCRIPT'
#!/usr/bin/env bash
printf 'demo-ok\n'
SCRIPT
chmod 0755 %{buildroot}/usr/bin/lpic-demo
%files
/usr/bin/lpic-demo
EOF
rpmbuild --define '_topdir /workspace/rpmbuild' -bb /workspace/rpmbuild/SPECS/lpic-helper.spec >/dev/null
rpmbuild --define '_topdir /workspace/rpmbuild' -bb /workspace/rpmbuild/SPECS/lpic-demo.spec >/dev/null
cp /workspace/rpmbuild/RPMS/noarch/*.rpm /workspace/repo/
createrepo_c /workspace/repo >/dev/null
cat > /etc/yum.repos.d/lpic-local.repo <<'EOF'
[lpic-local]
name=LPIC Daily local
baseurl=file:///workspace/repo
enabled=1
gpgcheck=0
EOF
dnf --disablerepo='*' --enablerepo=lpic-local makecache -y >/dev/null
chmod -R a+rwX /workspace
