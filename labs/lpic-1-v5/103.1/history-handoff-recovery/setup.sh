#!/usr/bin/env bash
set -euo pipefail
install -d -m 0700 /run/lpic
cat > /run/lpic/.bashrc <<'EOF'
export HISTFILE=/run/lpic/.bash_history
export HISTSIZE=25
export HISTFILESIZE=5
EOF
cat > /run/lpic/.bash_history <<'EOF'
pwd
export API_TOKEN=should-not-stay
echo old
EOF
