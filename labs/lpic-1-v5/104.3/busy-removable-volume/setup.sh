#!/usr/bin/env bash
set -euo pipefail
test -b /dev/vdb
mkfs.ext4 -F -q -L transferdisk /dev/vdb
mkdir -p /srv/transfer
mount /dev/vdb /srv/transfer
printf 'transfer-id=AB-5931\n' > /srv/transfer/manifest.txt
systemd-run --unit=lpic-transfer-reader.service --property=WorkingDirectory=/srv/transfer -- /usr/bin/sleep infinity
for _ in $(seq 1 30); do systemctl is-active --quiet lpic-transfer-reader.service && break; sleep 0.1; done
systemctl is-active --quiet lpic-transfer-reader.service
if umount /srv/transfer >/dev/null 2>&1; then
  echo 'expected busy mount did not hold' >&2
  exit 1
fi
