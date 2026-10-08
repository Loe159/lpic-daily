#!/usr/bin/env bash
set -euo pipefail
install -d -m 0755 /run/lpic
install -d -m 0755 /home/alice
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
chmod 0666 /home/alice/.bashrc /home/alice/.bash_history
touch /run/lpic/history-proof
chmod 0666 /run/lpic/history-proof
