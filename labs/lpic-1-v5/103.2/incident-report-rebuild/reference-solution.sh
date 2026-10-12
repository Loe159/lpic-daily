#!/usr/bin/env bash
set -euo pipefail
events=/workspace/input/events.tsv
owners=/workspace/input/owners.tsv
out=/workspace/output
awk -F '\t' '$3 == "ERROR" { print $2 }' "$events" | sort -u > "$out/error-hosts.raw"
tr '-' '_' < "$out/error-hosts.raw" > "$out/error-hosts.txt"
awk -F '\t' 'NR==FNR { owner[$1]=$2; next } { print $1 "|" owner[$1] }' "$owners" "$out/error-hosts.raw" > "$out/error-teams.txt"
split -l 2 -d -a 2 "$out/error-teams.txt" "$out/chunks/error-"
printf 'errors=%s\n' "$(awk -F '\t' '$3 == "ERROR" { n++ } END { print n+0 }' "$events")" > "$out/metrics.txt"
printf 'unique_hosts=%s\n' "$(wc -l < "$out/error-hosts.txt")" >> "$out/metrics.txt"
