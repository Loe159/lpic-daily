#!/usr/bin/env bash
set -euo pipefail

install -d -m 0755 /run/lpic
rm -f /run/lpic/worker-env

cat > /root/.bashrc <<'EOF'
REPORT_ENV=production
export REPORT_DEBUG=1
EOF

cat > /usr/local/bin/start-report-worker <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

{
  printf 'REPORT_ENV=%s\n' "${REPORT_ENV-<unset>}"
  printf 'REPORT_DEBUG=%s\n' "${REPORT_DEBUG-<unset>}"
} > /run/lpic/worker-env
EOF
chmod 0755 /usr/local/bin/start-report-worker
