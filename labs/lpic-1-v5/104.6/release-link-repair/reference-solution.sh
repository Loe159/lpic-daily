#!/usr/bin/env bash
set -euo pipefail
ln /workspace/releases/app-v2.conf /workspace/releases/archive-v2.conf
cp /workspace/releases/app-v2.conf /workspace/releases/snapshot-v2.conf
ln -sfn releases/app-v2.conf /workspace/current.conf
if ln /workspace/releases /workspace/dir-hardlink 2>/dev/null; then
  exit 1
fi
if ln /workspace/releases/app-v2.conf /run/lpic/cross-fs-link 2>/dev/null; then
  exit 1
fi
printf 'directory-hardlink=rejected\ncross-filesystem-hardlink=rejected\n' > /workspace/limitations.txt
