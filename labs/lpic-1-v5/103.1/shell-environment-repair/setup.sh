#!/usr/bin/env bash
set -euo pipefail

test -x /opt/lpic/approved/bin/report-status
test -x /opt/lpic/shadow/bin/report-status
install -d -m 0755 /run/lpic /var/lib/lpic/reports

cat > /root/.bash_profile <<'EOF'
export PATH="/opt/lpic/shadow/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin"
export REPORT_ENV="staging"
REPORT_FILE=/var/lib/lpic/reports/daily report.txt
export REPORT_FILE
EOF
