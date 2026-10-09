#!/usr/bin/env bash
set -euo pipefail
: > /workspace/audit/owners.tsv
for file in /usr/bin/bash /usr/bin/rpm /usr/bin/sed; do
  printf '%s\t%s\n' "$file" "$(rpm -qf --qf '%{NAME}\n' "$file")" >> /workspace/audit/owners.tsv
done
rpm -ql bash > /workspace/audit/bash-files.txt
