#!/usr/bin/env bash
set -euo pipefail
cat > /root/.bash_profile <<'EOF'
export PATH="/opt/lpic/approved/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin"
export REPORT_ENV="production"
export REPORT_FILE="/var/lib/lpic/reports/daily report.txt"
export HISTSIZE=3000
export HISTFILESIZE=6000
EOF
/usr/bin/bash -lc 'printf "%s\n%s\n" "$(uname -r)" "$(type -t report-status)" > /run/lpic/handoff-summary && report-status "$REPORT_FILE"'
