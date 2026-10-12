#!/usr/bin/env bash
set -euo pipefail
mkdir -p /workspace/input/config /workspace/output
printf 'alpha=1\n' > /workspace/input/config/a.conf
printf 'beta=2\n' > /workspace/input/config/b.conf
dd if=/dev/zero of=/workspace/input/disk-slice.bin bs=4096 count=1 status=none
printf 'LPIC' | dd of=/workspace/input/disk-slice.bin bs=1 seek=128 conv=notrunc status=none
printf 'release-one\n' | gzip -c > /workspace/input/payload.one
printf 'release-two\n' | bzip2 -c > /workspace/input/payload.two
printf 'release-three\n' | xz -c > /workspace/input/payload.three
