#!/usr/bin/env bash
set -euo pipefail
cat > /root/.bashrc <<'EOF'
export HISTFILE=/root/.bash_history
export HISTSIZE=2000
export HISTFILESIZE=4000
EOF
grep -v 'API_TOKEN' /root/.bash_history > /root/.bash_history.clean
mv /root/.bash_history.clean /root/.bash_history
cd /root
echo "$USER:$PWD" > /run/lpic/history-proof
/usr/bin/bash --noprofile --rcfile /root/.bashrc -ic 'history -c; history -r; history -s '''echo "$USER:$PWD" > /run/lpic/history-proof'''; history -w'
