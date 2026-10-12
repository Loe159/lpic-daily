#!/usr/bin/env bash
set -euo pipefail
modprobe brd rd_nr=1 rd_size=16384
udevadm settle
test -b /dev/ram0
mkfs.ext4 -F -q /dev/ram0
mount /dev/ram0 /run/lpic-ramcache
cp /var/lib/lpic-ramcache/source.txt /run/lpic-ramcache/source.txt
systemctl reset-failed lpic-ramcache-reader.service || true
systemctl start lpic-ramcache-reader.service
