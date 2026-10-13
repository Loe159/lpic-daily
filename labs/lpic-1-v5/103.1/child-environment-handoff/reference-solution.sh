#!/usr/bin/env bash
set -euo pipefail
cat > /run/lpic/worker-env-repair.sh <<'EOF'
export REPORT_ENV=production
unset REPORT_DEBUG
EOF
. /root/.bashrc
. /run/lpic/worker-env-repair.sh
/usr/local/bin/start-report-worker
