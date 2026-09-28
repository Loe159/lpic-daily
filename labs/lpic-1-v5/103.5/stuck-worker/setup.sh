#!/usr/bin/env bash
set -euo pipefail

install -d -m 0755 /run/lpic

cat > /usr/local/bin/lpic-signal-probe <<'EOF'
#!/usr/bin/env bash
set -u

record_resume() {
    printf 'continued\n' > /run/lpic/probe-continued

    own_pgid="$(ps -o pgid= -p "$$" | tr -d ' ')"
    terminal_pgid="$(ps -o tpgid= -p "$$" | tr -d ' ')"
    if [[ -n "$own_pgid" && -n "$terminal_pgid" && "$own_pgid" != "$terminal_pgid" ]]; then
        printf 'background\n' > /run/lpic/probe-background
    fi
}

trap 'printf "stopped\n" > /run/lpic/probe-stopped; kill -STOP "$$"' TSTP
trap 'record_resume' CONT
trap 'printf "terminated\n" > /run/lpic/probe-terminated; exit 0' TERM

while :; do
    sleep 2 &
    wait "$!"
done
EOF
chmod 0755 /usr/local/bin/lpic-signal-probe

nohup bash -c 'exec -a healthy-worker sleep infinity' >/dev/null 2>&1 &
printf '%s\n' "$!" > /run/lpic/healthy-worker.pid

nohup bash -c 'exec -a stuck-worker sleep infinity' >/dev/null 2>&1 &
printf '%s\n' "$!" > /run/lpic/stuck-worker.pid

sleep 0.1
