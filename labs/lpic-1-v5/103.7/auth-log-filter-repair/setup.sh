#!/usr/bin/env bash
set -euo pipefail
mkdir -p /workspace/input /workspace/output
cat > /workspace/input/auth.log <<'EOF'
2026-10-07 ssh user01 FAIL
2026-10-07 ssh admin FAIL
2026-10-07 sudo user22 FAIL
2026-10-07 ssh user7 FAIL
2026-10-07 ssh user99 FAIL
2026-10-07 ssh user44 OK
2026-10-07 ftp user33 FAIL
EOF
printf 'system ready\n' > /workspace/input/system.log
printf 'old auth\n' > /workspace/input/auth.log.bak
