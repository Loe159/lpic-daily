#!/usr/bin/env bash
set -euo pipefail

reload_pid="$(pgrep -f '^reload-worker' | head -n1)"
test -n "$reload_pid"
ps -o pid=,ppid=,stat=,comm=,args= -p "$reload_pid" > /run/lpic/maintenance-inspection
kill -USR1 "$reload_pid"
for _ in $(seq 1 50); do [[ -f /run/lpic/reload-requested ]] && break; sleep 0.02; done
test -f /run/lpic/reload-requested

/usr/local/bin/lpic-maintenance-job-probe &
probe_pid="$!"
sleep 0.1
kill -TSTP "$probe_pid"
for _ in $(seq 1 50); do [[ -f /run/lpic/maintenance-probe-stopped ]] && break; sleep 0.02; done
test -f /run/lpic/maintenance-probe-stopped
kill -CONT "$probe_pid"
for _ in $(seq 1 50); do [[ -f /run/lpic/maintenance-probe-background ]] && break; sleep 0.02; done
test -f /run/lpic/maintenance-probe-background
kill -TERM "$probe_pid"
wait "$probe_pid" || true

bash -c 'exec -a batch-worker sleep infinity' &
nohup bash -c 'exec -a handoff-daemon sleep infinity' >/run/lpic/handoff-daemon.log 2>&1 &
tmux new-session -d -s maintenance-ops 'sleep infinity'
