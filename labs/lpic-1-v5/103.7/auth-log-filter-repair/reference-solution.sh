#!/usr/bin/env bash
set -euo pipefail
grep -E '^2026-10-07 (ssh|sudo) user[[:digit:]]{2} FAIL$' /workspace/input/auth.log > /workspace/output/suspicious.txt
sed 's/ FAIL$/ DENIED/' /workspace/output/suspicious.txt > /workspace/output/normalized.txt
printf '%s\n' /workspace/input/*.log | sed 's#.*/##' | sort > /workspace/output/active-logs.txt
