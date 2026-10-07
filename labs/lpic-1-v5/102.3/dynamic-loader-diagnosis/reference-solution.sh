#!/usr/bin/env bash
set -euo pipefail
cd /workspace
ldd /usr/bin/true > ldd.txt
ldconfig -vN > search-paths.txt 2>&1 || true
ldconfig -p | grep 'libc\.so\.6' > cache-libc.txt
LD_LIBRARY_PATH=/workspace/lib ldd /usr/bin/true > override.txt
ldd /workspace/broken > missing.txt || true
