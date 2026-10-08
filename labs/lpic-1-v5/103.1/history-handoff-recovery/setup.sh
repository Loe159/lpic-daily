#!/usr/bin/env bash
set -euo pipefail
install -d -m 0777 /run/lpic
runuser -u alice -- /usr/bin/bash -eu <<'ALICE'
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
ALICE
