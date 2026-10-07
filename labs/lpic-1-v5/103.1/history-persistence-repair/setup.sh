#!/usr/bin/env bash
set -euo pipefail
install -d -m 0755 /run/lpic
rm -rf /root/.bash_history
cat > /root/.bash_profile <<'EOF'
export HISTFILE=/run/lpic/transient-history
export HISTSIZE=1
export HISTFILESIZE=1
EOF
cat > /root/.bash_history <<'EOF'
uname -a
rm -rf /tmp/reports
pwd
EOF
