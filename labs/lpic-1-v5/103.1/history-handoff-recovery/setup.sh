#!/usr/bin/env bash
set -euo pipefail
install -d -m 0755 /run/lpic
if [ -e /root/.bash_history ] || [ -L /root/.bash_history ]; then
    mv -f /root/.bash_history /root/.bash_history.image
fi
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
