#!/usr/bin/env bash
set -euo pipefail
renice 10 -p "$(cat /run/lpic/reindex-worker.pid)" >/dev/null
nice -n 15 bash -c 'exec -a batch-worker sleep infinity' &
sleep 0.1
service_pid="$(cat /run/lpic/service-worker.pid)"
reindex_pid="$(cat /run/lpic/reindex-worker.pid)"
batch_pid="$(pgrep -f '^batch-worker' | head -n1)"
{
  printf 'service-worker %s\n' "$(ps -o ni= -p "$service_pid" | tr -d ' ')"
  printf 'reindex-worker %s\n' "$(ps -o ni= -p "$reindex_pid" | tr -d ' ')"
  printf 'batch-worker %s\n' "$(ps -o ni= -p "$batch_pid" | tr -d ' ')"
} > /run/lpic/priority-report
