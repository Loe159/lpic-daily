#!/usr/bin/env bash
set -euo pipefail
install -d -m 0755 /run/lpic /var/lib/lpic/reports
cat > /root/.bashrc <<'EOF'
export PATH="/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin"
export REPORT_ENV=production
export REPORT_FILE="/var/lib/lpic/reports/daily report.txt"
EOF
rm -f /run/lpic/report-ok /run/lpic/external-command-summary
