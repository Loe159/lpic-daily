#!/usr/bin/env bash
set -euo pipefail
mkdir -p /run/lpic
bash -c 'exec -a service-worker sleep infinity' &
printf '%s\n' "$!" > /run/lpic/service-worker.pid
bash -c 'exec -a reindex-worker sleep infinity' &
printf '%s\n' "$!" > /run/lpic/reindex-worker.pid
rm -f /run/lpic/priority-report
