#!/usr/bin/env bash
set -euo pipefail
test -b /dev/vdb
mkfs.ext4 -F -q -N 128 -L alert-spool /dev/vdb
mkdir -p /srv/lpic-spool
mount /dev/vdb /srv/lpic-spool
mkdir -p /srv/lpic-spool/ack/stale /srv/lpic-spool/ack/recent
printf 'service=collector\n' > /srv/lpic-spool/README.txt
for n in 1 2 3; do printf 'ack=current-%s\n' "$n" > "/srv/lpic-spool/ack/recent/current-$n.ack"; done
for n in $(seq 1 2048); do
    if ! touch "/srv/lpic-spool/ack/stale/expired-$n.ack" 2>/dev/null; then break; fi
done
df -Pi /srv/lpic-spool | awk 'END {exit !($4 == 0)}'
