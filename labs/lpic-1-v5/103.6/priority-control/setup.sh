#!/usr/bin/env bash
set -euo pipefail
install -d -m 0777 /run/lpic
rm -f /run/lpic/default.pid /run/lpic/nice-default.pid /run/lpic/explicit.pid /run/lpic/observed.txt
nohup sleep 3600 >/dev/null 2>&1 &
printf '%s\n' "$!" > /run/lpic/base.pid
chmod -R a+rwX /run/lpic
