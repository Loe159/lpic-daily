#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' 'Maintenance du stockage a 18h : terminer les operations en cours.' | wall
sleep 1
grep -Fq 'Maintenance du stockage a 18h : terminer les operations en cours.' /var/lib/lpic-wall/received-a.log
grep -Fq 'Maintenance du stockage a 18h : terminer les operations en cours.' /var/lib/lpic-wall/received-b.log
