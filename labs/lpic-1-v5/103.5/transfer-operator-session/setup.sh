#!/usr/bin/env bash
set -euo pipefail

install -d -m 0755 /run/lpic

cat > /usr/local/bin/lpic-transfer-job-probe <<'EOF'
#!/usr/bin/env bash
set -u

record_resume() {
    printf 'continued\n' > /run/lpic/transfer-probe-continued

    own_pgid="$(ps -o pgid= -p "$$" | tr -d ' ')"
    terminal_pgid="$(ps -o tpgid= -p "$$" | tr -d ' ')"
    if [[ -n "$own_pgid" && -n "$terminal_pgid" && "$own_pgid" != "$terminal_pgid" ]]; then
        printf 'background\n' > /run/lpic/transfer-probe-background
    fi
}

trap 'printf "stopped\n" > /run/lpic/transfer-probe-stopped; kill -STOP "$$"' TSTP
trap 'record_resume' CONT
trap 'printf "terminated\n" > /run/lpic/transfer-probe-terminated; exit 0' TERM

while :; do
    sleep 2 &
    wait "$!"
done
EOF
chmod 0755 /usr/local/bin/lpic-transfer-job-probe

nohup bash -c 'exec -a queue-worker sleep infinity' >/dev/null 2>&1 &
printf '%s\n' "$!" > /run/lpic/queue-worker.pid

nohup bash -c 'exec -a leaky-worker sleep infinity' >/dev/null 2>&1 &
printf '%s\n' "$!" > /run/lpic/leaky-worker.pid

sleep 0.1
