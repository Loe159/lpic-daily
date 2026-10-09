#!/usr/bin/env bash
set -euo pipefail
test -b /dev/vdb
test -z "$(blkid -s TYPE -o value /dev/vdb 2>/dev/null || true)"
if findmnt --mountpoint /home >/dev/null 2>&1; then
  echo '/home must not be a separate mount in this VM image' >&2
  exit 1
fi
mkdir -p /home/lpic-training /var/log/lpic
if findmnt --mountpoint /var/log/lpic >/dev/null 2>&1; then
  echo 'log directory is already a mount' >&2
  exit 1
fi
printf 'account=retain-54\n' > /home/lpic-training/account.txt
printf 'batch=retain-19\n' > /var/log/lpic/records.log
