#!/usr/bin/env bash
set -euo pipefail
install -d -m 0755 /run/lpic /var/lib/lpic/reports
rm -f /run/lpic/handoff-summary /run/lpic/handoff-ready
cat > /root/.bashrc <<'EOF'
export PATH="/usr/local/sbin:/opt/lpic/shadow/bin:/usr/bin"
export HANDOFF_ROLE="secondary"
export HANDOFF_FILE="/tmp/handoff.txt"
export HISTSIZE=25
export HISTFILESIZE=50
export HISTCONTROL=ignorespace
EOF
