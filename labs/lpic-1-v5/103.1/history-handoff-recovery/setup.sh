#!/usr/bin/env bash
set -euo pipefail
mkdir -p /run/lpic
printf '%s\n' \
  'export HISTFILE=/root/.bash_history' \
  'export HISTSIZE=25' \
  'export HISTFILESIZE=5' > /root/.bashrc
printf '%s\n' \
  'pwd' \
  'export API_TOKEN=should-not-stay' \
  'echo old' > /root/.bash_history
rm -f /run/lpic/history-proof
