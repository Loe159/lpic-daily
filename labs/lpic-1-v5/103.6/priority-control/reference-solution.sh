#!/usr/bin/env bash
set -euo pipefail
nohup sleep 3600 >/dev/null 2>&1 & echo $! > /run/lpic/default.pid
nohup nice sleep 3600 >/dev/null 2>&1 & echo $! > /run/lpic/nice-default.pid
nohup nice -n 7 sleep 3600 >/dev/null 2>&1 & echo $! > /run/lpic/explicit.pid
renice 15 -p "$(cat /run/lpic/base.pid)" >/dev/null
ps -o ni= -p "$(cat /run/lpic/base.pid)" | tr -d ' ' > /run/lpic/observed.txt
