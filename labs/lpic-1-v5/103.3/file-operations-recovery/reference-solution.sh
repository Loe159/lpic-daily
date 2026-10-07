#!/usr/bin/env bash
set -euo pipefail
cd /workspace
mkdir -p work found
cp source/a.txt work/copied.txt
mv source/b.txt work/moved.txt
rm source/remove.me
printf '%s\n' logs/*.log | sed 's#logs/##' | sort > logs.list
find search -type f -size +50c -printf '%f\n' | sort > found-large.txt
find source -type f -name '*.txt' -exec cp '{}' found/ ';'
tar -cf bundle.tar -C archive-src .
(cd archive-src && find . -type f -print | sort | cpio -o > ../bundle.cpio 2>/dev/null)
dd if=raw.bin of=raw-copy.bin bs=8 status=none
gzip -c source/a.txt > a.txt.gz
bzip2 -c source/a.txt > a.txt.bz2
xz -c source/a.txt > a.txt.xz
file -b source/a.txt > type.txt
