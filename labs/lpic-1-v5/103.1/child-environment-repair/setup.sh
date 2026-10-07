#!/usr/bin/env bash
set -euo pipefail
install -d -m 0755 /run/lpic
rm -f /run/lpic/env-audit
cat > /root/.bash_profile <<'EOF'
export SHELL_NOTE="operator note"
DEPLOY_ENV=staging
export STALE_TOKEN="legacy-secret"
EOF
