#!/usr/bin/env bash
set -euo pipefail
mkdir -p /workspace/output
tar -cf /workspace/output/config.tar -C /workspace/input config/a.conf config/b.conf
(
  cd /workspace/input
  printf 'config/a.conf\nconfig/b.conf\n' | cpio -o -H newc > /workspace/output/config.cpio 2>/dev/null
)
dd if=/workspace/input/disk-slice.bin of=/workspace/output/disk-slice.copy bs=512 status=none
: > /workspace/output/recovered.txt
for source in /workspace/input/payload.one /workspace/input/payload.two /workspace/input/payload.three; do
  mime="$(file -b --mime-type "$source")"
  case "$mime" in
    application/gzip)
      cp "$source" /workspace/output/payload.gz
      zcat "$source" >> /workspace/output/recovered.txt
      ;;
    application/x-bzip2)
      cp "$source" /workspace/output/payload.bz2
      bzcat "$source" >> /workspace/output/recovered.txt
      ;;
    application/x-xz)
      cp "$source" /workspace/output/payload.xz
      xzcat "$source" >> /workspace/output/recovered.txt
      ;;
    *)
      exit 1
      ;;
  esac
done
