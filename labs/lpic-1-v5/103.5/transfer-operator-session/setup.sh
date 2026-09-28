#!/usr/bin/env bash
set -euo pipefail
install -d -m 0755 /run/lpic
cat > /usr/local/bin/lpic-term-probe <<'EOF'
#!/usr/bin/env bash
set -u
trap 'printf "terminated\n" > /run/lpic/term-probe-terminated; exit 0' TERM
while :; do sleep 2 & wait "$!"; done
EOF
chmod 0755 /usr/local/bin/lpic-term-probe
nohup bash -c 'exec -a queue-worker sleep infinity' >/dev/null 2>&1 &
printf '%s\n' "$!" > /run/lpic/queue-worker.pid
nohup bash -c 'exec -a leaky-worker sleep infinity' >/dev/null 2>&1 &
printf '%s\n' "$!" > /run/lpic/leaky-worker.pid
nohup /usr/local/bin/lpic-term-probe >/dev/null 2>&1 &
printf '%s\n' "$!" > /run/lpic/term-probe.pid
sleep 0.1
