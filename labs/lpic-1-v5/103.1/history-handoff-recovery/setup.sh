#!/usr/bin/env bash
set -euo pipefail
install -d -m 0755 /run/lpic
cat > /root/.bashrc <<'EOF'
export HISTFILE=/root/.bash_history
export HISTSIZE=25
export HISTFILESIZE=5
EOF
cat > /root/.bash_history <<'EOF'
pwd
export API_TOKEN=should-not-stay
echo old
EOF
rm -f /run/lpic/history-proof
