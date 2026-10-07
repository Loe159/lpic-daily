#!/usr/bin/env bash
set -euo pipefail
cat > /root/.bash_profile <<'EOF'
SHELL_NOTE="operator note"
export DEPLOY_ENV="production"
unset STALE_TOKEN
EOF
/usr/bin/bash -lc 'echo "deploy=$DEPLOY_ENV note=$SHELL_NOTE" > /run/lpic/env-audit'
