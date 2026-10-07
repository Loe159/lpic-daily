#!/usr/bin/env bash
set -euo pipefail
rm -rf /workspace/*
mkdir -p /workspace/repo
cp -a /opt/lpic/rpmrepo/. /workspace/repo/
cat > /etc/yum.repos.d/lpic-local.repo <<'EOF'
[lpic-local]
name=LPIC Daily local
baseurl=file:///workspace/repo
enabled=1
gpgcheck=0
EOF
dnf --disablerepo='*' --enablerepo=lpic-local makecache -y >/dev/null
chmod -R a+rwX /workspace
