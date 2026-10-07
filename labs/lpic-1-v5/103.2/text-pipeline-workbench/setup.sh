#!/usr/bin/env bash
set -euo pipefail
rm -rf /workspace/*
cat > /workspace/input.txt <<'EOF'
alpha:3
beta:1
alpha:2
gamma:2
EOF
printf 'red\nblue\n' > /workspace/left.txt
printf '1\n2\n' > /workspace/right.txt
gzip -c /workspace/input.txt > /workspace/input.txt.gz
bzip2 -c /workspace/input.txt > /workspace/input.txt.bz2
xz -c /workspace/input.txt > /workspace/input.txt.xz
chmod -R a+rwX /workspace
