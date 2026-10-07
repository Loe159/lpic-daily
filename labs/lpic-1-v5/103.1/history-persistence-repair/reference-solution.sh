#!/usr/bin/env bash
set -euo pipefail
cat > /root/.bash_profile <<'EOF'
export HISTFILE=/root/.bash_history
export HISTSIZE=1000
export HISTFILESIZE=2000
EOF
cat > /root/.bash_history <<'EOF'
uname -a
pwd
echo maintenance-complete
EOF
