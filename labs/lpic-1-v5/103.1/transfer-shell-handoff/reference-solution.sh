#!/usr/bin/env bash
set -euo pipefail
cat > /root/.bashrc <<'EOF'
export PATH="/opt/lpic/approved/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin"
export HANDOFF_ROLE="primary"
export HANDOFF_FILE="/var/lib/lpic/reports/handoff notes.txt"
export HISTSIZE=1500
export HISTFILESIZE=3000
export HISTCONTROL=ignoredups
EOF
/usr/bin/bash --noprofile --rcfile /root/.bashrc -ic 'printf "%s\n%s\n" "$(uname -m)" "$(type -t report-status)" > /run/lpic/handoff-summary && printf "ready\n" > /run/lpic/handoff-ready'
