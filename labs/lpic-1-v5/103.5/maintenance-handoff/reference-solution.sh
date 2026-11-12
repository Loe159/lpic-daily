#!/usr/bin/env bash
set -Eeuo pipefail
/usr/local/bin/lpic-batch-worker >/run/lpic/batch-worker.log 2>&1 </dev/null &
nohup /usr/local/bin/lpic-handoff-daemon >/run/lpic/handoff-daemon.log 2>&1 </dev/null &
tmux new-session -d -s maintenance-ops
