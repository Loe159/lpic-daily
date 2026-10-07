#!/usr/bin/env bash
set -euo pipefail
cat > /workspace/config.txt <<'EOF'
alpha=1
beta=new
copy=shared
copy=shared
anchor=before-end
EOF
cat > /root/.bash_profile <<'EOF'
export EDITOR=vim
EOF
printf '%s\n%s\n' "$(command -v vim)" "$(command -v nano)" > /workspace/editor-tools.txt
printf 'insert\nmove\nsearch\noperator:y\noperator:d\ncommand\nwrite\n' > /run/lpic/vim-events
