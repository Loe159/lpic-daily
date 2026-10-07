#!/usr/bin/env bash
set -euo pipefail
rm -rf /workspace/*
mkdir -p /workspace/repo /workspace/root /workspace/cache
cp -a /opt/lpic/rpmrepo/. /workspace/repo/
cat > /workspace/lpic-local.repo <<'EOF'
[lpic-local]
name=LPIC Daily local
baseurl=file:///workspace/repo
enabled=1
gpgcheck=0
EOF
rpm --root /workspace/root --initdb
chmod -R a+rwX /workspace
