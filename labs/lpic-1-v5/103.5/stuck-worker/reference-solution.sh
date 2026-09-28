#!/usr/bin/env bash
set -euo pipefail

kill -TERM "$(cat /run/lpic/stuck-worker.pid)"

/usr/local/bin/lpic-signal-probe &
probe_pid="$!"

sleep 0.1
kill -TSTP "$probe_pid"
kill -CONT "$probe_pid"
sleep 0.2
kill -TERM "$probe_pid"
wait "$probe_pid" || true
