#!/usr/bin/env bash
set -euo pipefail

parted --script /dev/vdb mklabel gpt
parted --script /dev/vdb mkpart data ext4 1MiB 385MiB
parted --script /dev/vdb mkpart swap linux-swap 385MiB 100%
partprobe /dev/vdb

mkfs.ext4 -F /dev/vdb1
mkswap /dev/vdb2
