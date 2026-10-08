#!/usr/bin/env bash
set -euo pipefail
runuser -u alice -- /usr/bin/bash -eu <<'ALICE'
cat > /home/alice/.bashrc <<'EOF'
export HISTFILE=/home/alice/.bash_history
export HISTSIZE=2000
export HISTFILESIZE=4000
EOF
grep -v 'API_TOKEN' /home/alice/.bash_history > /home/alice/.bash_history.clean
mv /home/alice/.bash_history.clean /home/alice/.bash_history
cat > /home/alice/.lpic-history-update <<'EOF'
cd /home/alice
echo "$USER:$PWD" > /run/lpic/history-proof
history -c
history -r
history -s 'echo "$USER:$PWD" > /run/lpic/history-proof'
history -w
EOF
HOME=/home/alice /usr/bin/bash --noprofile --rcfile /home/alice/.bashrc -ic 'source /home/alice/.lpic-history-update'
ALICE
