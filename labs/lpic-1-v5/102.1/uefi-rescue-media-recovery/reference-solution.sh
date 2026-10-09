#!/usr/bin/env bash
set -euo pipefail
parted --script /dev/vdb set 1 esp on
partprobe /dev/vdb
mount /dev/vdb1 /mnt/lpic-rescue-esp
