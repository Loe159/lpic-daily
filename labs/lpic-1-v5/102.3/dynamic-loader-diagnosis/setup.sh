#!/usr/bin/env bash
set -euo pipefail
rm -rf /workspace/*
mkdir -p /workspace/lib
cp /lib64/libc.so.6 /workspace/lib/libc.so.6
cp /usr/bin/true /workspace/broken
patchelf --add-needed liblpic_missing.so /workspace/broken
chmod -R a+rwX /workspace
