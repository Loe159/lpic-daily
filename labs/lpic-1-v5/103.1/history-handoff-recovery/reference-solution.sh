#!/usr/bin/env bash
set -euo pipefail
cat > /run/lpic/.bashrc <<'EOF'
export HISTFILE=/run/lpic/.bash_history
export HISTSIZE=2000
export HISTFILESIZE=4000
EOF
/usr/bin/grep -v 'API_TOKEN' /run/lpic/.bash_history > /run/lpic/.bash_history.clean
mv /run/lpic/.bash_history.clean /run/lpic/.bash_history
HOME=/run/lpic USER=root /usr/bin/bash --noprofile --norc -c '
  set -e
  . /run/lpic/.bashrc
  set -o history
  history -c
  history -r
  cd /run/lpic
  echo "$USER:$PWD"
  history -s '\''echo "$USER:$PWD"'\''
  history -w
  HISTFILE=/dev/null
'
