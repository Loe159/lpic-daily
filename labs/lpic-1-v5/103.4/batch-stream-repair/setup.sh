#!/usr/bin/env bash
set -euo pipefail
mkdir -p /workspace/input /workspace/output /workspace/bin
cat > /workspace/input/jobs.txt <<'EOF'
alpha
bad-beta
gamma
bad-delta
EOF
cat > /workspace/bin/audit-stream <<'EOF'
#!/usr/bin/env bash
set -eu
while IFS= read -r job; do
  case "$job" in
    bad-*) printf 'ERR:%s\n' "$job" >&2 ;;
    *) printf 'OK:%s\n' "$job" ;;
  esac
done
EOF
chmod 0755 /workspace/bin/audit-stream
printf 'STALE\n' > /workspace/output/stdout.log
printf 'previous-run\n' > /workspace/output/history.log
