#!/usr/bin/env bash
set -euo pipefail
cat > /root/.bashrc <<'EOF'
export SERVICE_MODE=production
LOCAL_NOTE=operator-note
unset LEGACY_TOKEN
EOF
/usr/bin/bash --noprofile --rcfile /root/.bashrc -ic 'printf "%s|%s\n" "$SERVICE_MODE" "$LOCAL_NOTE" > /run/lpic/env-summary'
