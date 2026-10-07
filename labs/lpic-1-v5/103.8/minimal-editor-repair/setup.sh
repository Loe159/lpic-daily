#!/usr/bin/env bash
set -euo pipefail
mkdir -p /workspace
cat > /root/.bashrc <<'EOF'
export EDITOR=nano
EOF
cat > /workspace/app.conf <<'EOF'
[client]
timeout=30

[server]
port=8080
mode=debug
obsolete=true
audit=enabled
EOF
rm -f /workspace/editor-inventory
