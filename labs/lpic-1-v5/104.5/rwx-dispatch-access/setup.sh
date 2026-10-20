#!/usr/bin/env bash
set -euo pipefail
id alice >/dev/null
install -d -o root -g root -m 0755 /workspace/dispatch /workspace/dispatch/bin /workspace/dispatch/reports
cat > /workspace/dispatch/bin/prepare.sh <<'SCRIPT'
#!/bin/sh
printf 'dispatch-ready\n'
SCRIPT
printf 'Date,Service,Etat\n2026-10-20,dispatch,ready\n' > /workspace/dispatch/reports/daily.txt
chmod 0644 /workspace/dispatch/bin/prepare.sh
chmod 0666 /workspace/dispatch/reports/daily.txt
chmod 0744 /workspace/dispatch/reports
