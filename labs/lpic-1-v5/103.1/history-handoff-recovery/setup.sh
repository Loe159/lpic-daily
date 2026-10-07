#!/usr/bin/env bash
set -euo pipefail
install -d -m 0755 /run/lpic
install -d -m 0700 -o alice -g alice /home/alice
cat > /home/alice/.bashrc <<'EOF'
export HISTFILE=/home/alice/.bash_history
export HISTSIZE=25
export HISTFILESIZE=5
EOF
cat > /home/alice/.bash_history <<'EOF'
pwd
export API_TOKEN=should-not-stay
echo old
EOF
chown alice:alice /home/alice/.bashrc /home/alice/.bash_history
touch /run/lpic/history-proof
chown alice:alice /run/lpic/history-proof
