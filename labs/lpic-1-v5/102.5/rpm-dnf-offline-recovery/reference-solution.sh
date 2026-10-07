#!/usr/bin/env bash
set -euo pipefail
cd /workspace
demo=$(printf '%s\n' repo/lpic-demo-*.rpm)
helper=$(printf '%s\n' repo/lpic-helper-*.rpm)
rpm -qpi "$demo" > query.txt
rpm2cpio "$demo" | cpio -t > payload.txt
rpm -Kv "$demo" > package-check.txt 2>&1
rpm --root /workspace/root -i "$helper"
rpm --root /workspace/root -i "$demo"
rpm --root /workspace/root -ql lpic-demo > files.txt
rpm --root /workspace/root -qf /usr/bin/lpic-demo > owner.txt
printf '#tampered\n' >> /workspace/root/usr/bin/lpic-demo
rpm --root /workspace/root -V lpic-demo > verify-broken.txt || true
rpm --root /workspace/root -U --replacepkgs "$demo"
rpm --root /workspace/root -V lpic-demo > verify-clean.txt
rpm --root /workspace/root -e lpic-demo
rpm --root /workspace/root -e lpic-helper
dnf --installroot=/workspace/root --releasever=44 \
  --setopt=reposdir=/workspace --setopt=cachedir=/workspace/cache \
  --disablerepo='*' --enablerepo=lpic-local install -y lpic-demo >/dev/null
dnf --installroot=/workspace/root --releasever=44 \
  --setopt=reposdir=/workspace --setopt=cachedir=/workspace/cache \
  --disablerepo='*' --enablerepo=lpic-local repolist > repos.txt
