#!/usr/bin/env bash
set -euo pipefail
command -v rpmbuild >/dev/null
command -v gpg >/dev/null
command -v gpgv >/dev/null
command -v rpm >/dev/null
install -d -m 0755 /srv/lpic-recovery /etc/lpic-recovery
install -d -m 0700 /var/lib/lpic-daily/rpm-signing
install -d -m 0755 /tmp/lpic-ledger-rpmbuild/{BUILD,RPMS,SOURCES,SPECS,SRPMS}
cat > /tmp/lpic-ledger-rpmbuild/SPECS/ledger.spec <<'EOF'
Name: lpic-ledger-watch
Version: 1.0
Release: 1
Summary: LPIC exercise telemetry
License: MIT
BuildArch: noarch
%description
Offline LPIC RPM integrity practice fixture.
%install
mkdir -p %{buildroot}/usr/local/bin
cat > %{buildroot}/usr/local/bin/lpic-ledger-watch <<'LPIC_AGENT'
#!/bin/sh
printf 'ledger=ready\n'
LPIC_AGENT
chmod 0755 %{buildroot}/usr/local/bin/lpic-ledger-watch
%files
/usr/local/bin/lpic-ledger-watch
EOF
rpmbuild --define '_topdir /tmp/lpic-ledger-rpmbuild' -bb /tmp/lpic-ledger-rpmbuild/SPECS/ledger.spec >/dev/null
good=/srv/lpic-recovery/lpic-ledger-watch-1.0-1.noarch.rpm
cp /tmp/lpic-ledger-rpmbuild/RPMS/noarch/lpic-ledger-watch-1.0-1.noarch.rpm "$good"
export GNUPGHOME=/var/lib/lpic-daily/rpm-signing
gpg --batch --pinentry-mode loopback --passphrase '' --quick-generate-key 'LPIC Offline Archive <archive@lpic.invalid>' ed25519 sign 0 >/dev/null 2>&1
gpg --armor --batch --yes --detach-sign --output "$good.asc" "$good"
gpg --export > /etc/lpic-recovery/trustedkeys.gpg
gpgv --keyring /etc/lpic-recovery/trustedkeys.gpg "$good.asc" "$good" >/dev/null 2>&1
rpm -K --nosignature "$good" | grep -q 'digests OK'
cp "$good" /srv/lpic-recovery/lpic-ledger-watch-damaged.rpm
bad=/srv/lpic-recovery/lpic-ledger-watch-damaged.rpm
size=$(stat -c %s "$bad")
printf 'DAMAGED-ARCHIVE' | dd of="$bad" bs=1 seek="$((size - 55))" conv=notrunc status=none
if rpm -K --nosignature "$bad" >/dev/null 2>&1; then
  echo 'Expected damaged archive digest failure' >&2
  exit 1
fi
rpm -Uvh --replacepkgs "$good" >/dev/null
printf '#!/bin/sh\nprintf "ledger=corrupt\\n"\n' > /usr/local/bin/lpic-ledger-watch
chmod 0755 /usr/local/bin/lpic-ledger-watch
if rpm -V lpic-ledger-watch >/dev/null 2>&1; then
  echo 'Expected installed file corruption' >&2
  exit 1
fi
