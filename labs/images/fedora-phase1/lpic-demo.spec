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
