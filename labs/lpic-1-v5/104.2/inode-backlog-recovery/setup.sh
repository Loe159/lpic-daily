#!/usr/bin/env bash
set -euo pipefail
mkdir -p /workspace/spool/ack/stale /workspace/spool/ack/recent /workspace/reports /workspace/archive
for n in $(seq 1 160); do printf 'ack=expired-%s\n' "$n" > "/workspace/spool/ack/stale/expired-$n.ack"; done
for n in 1 2 3; do printf 'ack=current-%s\n' "$n" > "/workspace/spool/ack/recent/current-$n.ack"; done
printf 'report=production\n' > /workspace/reports/current.txt
printf 'do-not-delete=reference-archive\n' > /workspace/archive/reference.tar.note
head -c 65536 /dev/zero > /workspace/archive/reference.bin
