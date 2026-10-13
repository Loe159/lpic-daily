#!/usr/bin/env bash
set -euo pipefail
cat > /run/lpic/repair-environment.sh <<'EOF'
export SERVICE_MODE=production
export -n LOCAL_NOTE
unset LEGACY_TOKEN
printf '%s|%s\n' "$SERVICE_MODE" "$LOCAL_NOTE" > /run/lpic/env-summary
EOF
