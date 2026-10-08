#!/usr/bin/env bash
set -euo pipefail
cat > /root/.bashrc <<'EOF'
export HISTFILE=/root/.bash_history
export HISTSIZE=2000
export HISTFILESIZE=4000
EOF
grep -v 'API_TOKEN' /root/.bash_history > /root/.bash_history.clean
mv /root/.bash_history.clean /root/.bash_history
cat > /root/.lpic-history-update <<'EOF'
cd /root
echo "$USER:$PWD" > /run/lpic/history-proof
history -c
history -r
history -s 'echo "$USER:$PWD" > /run/lpic/history-proof'
history -w
EOF
HOME=/root USER=root /usr/bin/bash --noprofile --rcfile /root/.bashrc -ic 'source /root/.lpic-history-update'
