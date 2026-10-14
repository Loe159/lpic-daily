#!/usr/bin/env bash
set -euo pipefail
cat > /run/lpic/run-approved-report.sh <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
tool=/opt/lpic/approved/bin/report-status
"$tool" "$REPORT_FILE"
printf '%s\nready\n' "$tool" > /run/lpic/external-command-summary
EOF
chmod 0755 /run/lpic/run-approved-report.sh
