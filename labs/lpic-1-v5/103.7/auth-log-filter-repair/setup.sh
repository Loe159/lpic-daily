#!/usr/bin/env bash
set -euo pipefail
mkdir -p /workspace/input /workspace/output
# Per-session randomized valid records make fixed-output submissions fail.
seed="$(od -An -N2 -tu2 /dev/urandom | tr -d '[:space:]')"
a="$(printf '%02d' "$((seed % 100))")"
b="$(printf '%02d' "$(((seed / 11 + 37) % 100))")"
c="$(printf '%02d' "$(((seed / 7 + 61) % 100))")"
cat > /workspace/input/auth.log <<EOF
2026-10-07 ssh user$a FAIL
2026-10-07 ssh admin FAIL
2026-10-07 sudo user$b FAIL
2026-10-07 ssh user7 FAIL
2026-10-07 ssh user$c FAIL
2026-10-07 ssh user44 OK
2026-10-07 ftp user33 FAIL
EOF
printf 'system ready\n' > /workspace/input/system.log
printf 'old auth\n' > /workspace/input/auth.log.bak
