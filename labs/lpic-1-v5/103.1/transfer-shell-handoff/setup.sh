#!/usr/bin/env bash
set -euo pipefail
install -d -m 0755 /run/lpic /var/lib/lpic/reports
rm -f /run/lpic/report-ok /run/lpic/handoff-summary
cat > /root/.bash_profile <<'EOF'
export PATH="/usr/local/sbin:/opt/lpic/shadow/bin:/usr/bin"
export REPORT_ENV="qa"
export REPORT_FILE="/tmp/report.txt"
export HISTSIZE=50
export HISTFILESIZE=50
EOF
