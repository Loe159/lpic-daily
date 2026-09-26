#!/usr/bin/env bash
set -euo pipefail

install -d -m 0755 /opt/lpic/approved/bin /opt/lpic/shadow/bin /run/lpic /var/lib/lpic/reports

cat > /opt/lpic/approved/bin/report-status <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

[[ "${REPORT_ENV:-}" == "production" ]] || exit 21
[[ "$#" -eq 1 ]] || exit 22
[[ "$1" == "/var/lib/lpic/reports/daily report.txt" ]] || exit 23

printf 'ready\n' > /run/lpic/report-ok
EOF
chmod 0755 /opt/lpic/approved/bin/report-status

cat > /opt/lpic/shadow/bin/report-status <<'EOF'
#!/usr/bin/env bash
exit 17
EOF
chmod 0755 /opt/lpic/shadow/bin/report-status

cat > /root/.bash_profile <<'EOF'
export PATH="/opt/lpic/shadow/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin"
export REPORT_ENV="staging"
REPORT_FILE=/var/lib/lpic/reports/daily report.txt
export REPORT_FILE
EOF
