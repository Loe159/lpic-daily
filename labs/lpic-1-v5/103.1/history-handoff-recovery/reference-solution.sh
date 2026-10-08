#!/usr/bin/env bash
set -euo pipefail
cat > /run/lpic/.bashrc <<'EOF'
export HISTFILE=/run/lpic/.bash_history
export HISTSIZE=2000
export HISTFILESIZE=4000
EOF
grep -v 'API_TOKEN' /run/lpic/.bash_history > /run/lpic/.bash_history.clean
mv /run/lpic/.bash_history.clean /run/lpic/.bash_history
cat > /run/lpic/.lpic-history-update <<'EOF'
cd /run/lpic
echo "$USER:$PWD" > /run/lpic/history-proof
history -c
history -r
history -s 'echo "$USER:$PWD" > /run/lpic/history-proof'
history -w
EOF
HOME=/run/lpic USER=root /usr/bin/bash --noprofile --rcfile /run/lpic/.bashrc -ic 'source /run/lpic/.lpic-history-update'
