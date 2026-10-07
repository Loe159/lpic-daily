#!/usr/bin/env bash
set -euo pipefail
mkdir -p /workspace/release
cp -R /workspace/source/app /workspace/release/app
mkdir -p /workspace/release/config /workspace/release/large-logs
mv /workspace/release/app/conf/*.conf /workspace/release/config/
rmdir /workspace/release/app/conf
find /workspace/release/app -type f -name '*.tmp' -mtime +7 -delete
find /workspace/release/app/logs -type f -size +20c -exec mv -t /workspace/release/large-logs -- {} +
touch /workspace/release/READY
