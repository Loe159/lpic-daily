#!/usr/bin/env bash
set -euo pipefail
test -x /opt/lpic/approved/bin/report-status
test -x /opt/lpic/shadow/bin/report-status
install -d -m 0755 /run/lpic /var/lib/lpic/reports
rm -f /run/lpic/report-ok /run/lpic/system-summary
cat > /root/.bash_profile <<'EOF'
export PATH="/opt/lpic/shadow/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin"
export REPORT_ENV="staging"
export REPORT_FILE="/var/lib/lpic/reports/daily report.txt.bak"
export HISTSIZE=100
export HISTFILESIZE=100
EOF
