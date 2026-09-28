#!/usr/bin/env bash
set -euo pipefail
kill -TERM "$(cat /run/lpic/leaky-worker.pid)"
kill -TERM "$(cat /run/lpic/term-probe.pid)"
for _ in $(seq 1 50); do [[ -f /run/lpic/term-probe-terminated ]] && break; sleep 0.02; done
test -f /run/lpic/term-probe-terminated
bash -c 'exec -a background-worker sleep infinity' &
nohup bash -c 'exec -a survivor-worker sleep infinity' >/run/lpic/survivor.log 2>&1 &
tmux new-session -d -s transfer-ops 'sleep infinity'
