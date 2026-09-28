#!/usr/bin/env bash
set -euo pipefail

ps -o pid=,ppid=,stat=,comm=,args= \
  -p "$(cat /run/lpic/queue-worker.pid),$(cat /run/lpic/leaky-worker.pid)" \
  > /run/lpic/transfer-process-inspection

leaky_pid="$(pgrep -f '^leaky-worker' | head -n1)"
test -n "$leaky_pid"
kill -TERM "$leaky_pid"

/usr/local/bin/lpic-transfer-job-probe &
probe_pid="$!"
sleep 0.1
kill -TSTP "$probe_pid"
for _ in $(seq 1 50); do [[ -f /run/lpic/transfer-probe-stopped ]] && break; sleep 0.02; done
test -f /run/lpic/transfer-probe-stopped
kill -CONT "$probe_pid"
for _ in $(seq 1 50); do [[ -f /run/lpic/transfer-probe-background ]] && break; sleep 0.02; done
test -f /run/lpic/transfer-probe-background
kill -TERM "$probe_pid"
wait "$probe_pid" || true

bash -c 'exec -a background-worker sleep infinity' &
nohup bash -c 'exec -a survivor-worker sleep infinity' >/run/lpic/survivor.log 2>&1 &
tmux new-session -d -s transfer-ops 'sleep infinity'
