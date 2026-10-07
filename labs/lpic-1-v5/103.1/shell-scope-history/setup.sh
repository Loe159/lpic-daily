#!/usr/bin/env bash
set -euo pipefail
rm -rf /workspace/* /run/lpic/*
mkdir -p /workspace/tools /run/lpic
cat > /workspace/tools/outside-tool <<'EOF'
#!/usr/bin/env bash
printf 'outside-ok\n'
EOF
chmod 0755 /workspace/tools/outside-tool
cat > /root/.bash_profile <<'EOF'
export PATH="/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin"
export HISTFILE=/root/.bash_history
export HISTSIZE=1000
export HISTFILESIZE=2000
EOF
: > /root/.bash_history
chmod -R a+rwX /workspace /run/lpic
