#!/usr/bin/env bash
set -euo pipefail

kill -TERM "$(cat /run/lpic/stuck-worker.pid)"

/usr/local/bin/lpic-signal-probe &
probe_pid="$!"

sleep 0.1
kill -TSTP "$probe_pid"

for _ in $(seq 1 50); do
    [[ -f /run/lpic/probe-stopped ]] && break
    sleep 0.02
done
test -f /run/lpic/probe-stopped

kill -CONT "$probe_pid"
for _ in $(seq 1 50); do
    [[ -f /run/lpic/probe-background ]] && break
    sleep 0.02
done
test -f /run/lpic/probe-background

kill -TERM "$probe_pid"
wait "$probe_pid" || true
