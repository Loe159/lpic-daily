#!/usr/bin/env bash
set -euo pipefail
rm -rf /workspace/*
mkdir -p /workspace/glob
cat > /workspace/corpus.txt <<'EOF'
alpha 123
beta 456
ALPHA 789
gamma abc
delta 42
EOF
: > /workspace/glob/report1.log
: > /workspace/glob/report22.log
: > /workspace/glob/reportA.log
: > /workspace/glob/note.txt
chmod -R a+rwX /workspace
