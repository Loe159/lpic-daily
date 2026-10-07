#!/usr/bin/env bash
set -euo pipefail
rm -rf /workspace/*
mkdir -p /workspace/source /workspace/logs /workspace/search /workspace/archive-src
printf 'A-source\n' > /workspace/source/a.txt
printf 'B-source\n' > /workspace/source/b.txt
printf 'delete-me\n' > /workspace/source/remove.me
printf 'app\n' > /workspace/logs/app.log
printf 'audit\n' > /workspace/logs/audit.log
printf 'ignore\n' > /workspace/logs/note.txt
printf 'tiny\n' > /workspace/search/small.dat
printf '%0128d\n' 0 > /workspace/search/large.dat
printf 'one\n' > /workspace/archive-src/one.txt
printf 'two\n' > /workspace/archive-src/two.txt
printf '0123456789abcdef0123456789abcdef\n' > /workspace/raw.bin
chmod -R a+rwX /workspace
