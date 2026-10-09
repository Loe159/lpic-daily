#!/usr/bin/env bash
set -euo pipefail
parted --script /dev/vdb mklabel gpt
parted --script /dev/vdb mkpart primary 1MiB 100%
partprobe /dev/vdb
for _ in $(seq 1 100); do test -b /dev/vdb1 && break; sleep 0.1; done
test -b /dev/vdb1
pvcreate -y /dev/vdb1
vgcreate lpicvg /dev/vdb1
lvcreate -L 256M -n data lpicvg
mkfs.ext4 -F -q /dev/lpicvg/data
mount /dev/lpicvg/data /srv/app-data
