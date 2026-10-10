#!/usr/bin/env bash
set -euo pipefail
tool=/workspace/bin/audit-stream
input=/workspace/input/jobs.txt
out=/workspace/output
"$tool" < "$input" > "$out/stdout.log" 2> "$out/stderr.log"
"$tool" < "$input" > "$out/combined.log" 2>&1
"$tool" < "$input" >> "$out/history.log" 2>/dev/null
"$tool" < "$input" 2>/dev/null | tee "$out/live.log" | wc -l > "$out/success-count.txt"
