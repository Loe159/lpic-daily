#!/usr/bin/env bash
set -euo pipefail
install -d -m 0755 /run/lpic
cat > /root/.bashrc <<'EOF'
SERVICE_MODE=staging
export LOCAL_NOTE=operator-note
export LEGACY_TOKEN=deprecated
EOF
rm -f /run/lpic/env-summary
