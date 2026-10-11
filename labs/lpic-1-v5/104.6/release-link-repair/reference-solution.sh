#!/usr/bin/env bash
set -euo pipefail
ln /workspace/releases/app-v2.conf /workspace/releases/archive-v2.conf
cp /workspace/releases/app-v2.conf /workspace/releases/snapshot-v2.conf
ln -sfn releases/app-v2.conf /workspace/current.conf
if ln /workspace/releases /workspace/dir-hardlink 2>/dev/null; then
  exit 1
fi
test "$(stat -c %d /workspace)" != "$(stat -c %d /dev/shm)"
if LC_ALL=C ln /workspace/releases/app-v2.conf /dev/shm/lpic-cross-fs-link 2> /workspace/.cross-fs-error; then
  exit 1
fi
grep -Fq 'Invalid cross-device link' /workspace/.cross-fs-error
rm -f /workspace/.cross-fs-error
printf 'directory-hardlink=rejected\ncross-filesystem-hardlink=rejected\n' > /workspace/limitations.txt
