#!/usr/bin/env bash
set -euo pipefail
cd /workspace
demo=$(printf '%s\n' repo/lpic-demo-*.rpm)
helper=$(printf '%s\n' repo/lpic-helper-*.rpm)
rpm -qpi "$demo" > query.txt
rpm2cpio "$demo" | cpio -t > payload.txt
rpm -Kv "$demo" > package-check.txt 2>&1
rpm -i "$helper"
rpm -i "$demo"
rpm -ql lpic-demo > files.txt
rpm -qf /usr/bin/lpic-demo > owner.txt
printf '#tampered\n' >> /usr/bin/lpic-demo
rpm -V lpic-demo > verify-broken.txt || true
rpm -U --replacepkgs "$demo"
rpm -V lpic-demo > verify-clean.txt
rpm -e lpic-demo
rpm -e lpic-helper
dnf --disablerepo='*' --enablerepo=lpic-local install -y lpic-demo >/dev/null
dnf --disablerepo='*' --enablerepo=lpic-local repolist > repos.txt
