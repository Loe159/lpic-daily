#!/usr/bin/env bash
set -euo pipefail
mkdir -p /workspace/input /workspace/output
cat > /workspace/input/events.tsv <<'EOF'
2026-10-07T10:03:00Z	web-02	ERROR	timeout
2026-10-07T10:01:00Z	web-01	WARN	cpu_high
2026-10-07T10:02:00Z	db-01	ERROR	disk_full
2026-10-07T10:04:00Z	web-02	ERROR	timeout
2026-10-07T10:05:00Z	api-01	INFO	ready
2026-10-07T10:06:00Z	web-01	ERROR	oom
EOF
cat > /workspace/input/owners.tsv <<'EOF'
api-01	platform
db-01	data
web-01	frontend
web-02	frontend
EOF
# Randomize an incident hostname so static answer files cannot pass across sessions.
suffix="$(od -An -N2 -tu2 /dev/urandom | tr -d '[:space:]')"
suffix="$(printf '%02d' "$((10 + suffix % 90))")"
sed -i "s/web-02/web-$suffix/g" /workspace/input/events.tsv /workspace/input/owners.tsv
mkdir -p /workspace/output/chunks
