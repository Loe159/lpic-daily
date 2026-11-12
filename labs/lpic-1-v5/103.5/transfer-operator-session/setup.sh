#!/usr/bin/env bash
set -euo pipefail
install -d -m 0755 /run/lpic
cat > /usr/local/bin/lpic-maintenance-job-probe <<'EOF'
#!/usr/bin/env bash
set -u
event() { printf '%s %s\n' "$$" "$1" >> /run/lpic/maintenance-probe-events; }
record_foreground() {
    local own_pgid terminal_pgid
    own_pgid="$(ps -o pgid= -p "$$" | tr -d ' ')"
    terminal_pgid="$(ps -o tpgid= -p "$$" | tr -d ' ')"
    if [[ -n "$own_pgid" && "$own_pgid" == "$terminal_pgid" ]]; then
        printf 'foreground\n' > /run/lpic/maintenance-probe-foreground
        event foreground
        return 0
    fi
    return 1
}
record_resume() {
    printf 'continued\n' > /run/lpic/maintenance-probe-continued
    local own_pgid terminal_pgid
    own_pgid="$(ps -o pgid= -p "$$" | tr -d ' ')"
    terminal_pgid="$(ps -o tpgid= -p "$$" | tr -d ' ')"
    if [[ -n "$own_pgid" && -n "$terminal_pgid" && "$own_pgid" != "$terminal_pgid" ]]; then
        printf 'background\n' > /run/lpic/maintenance-probe-background
        event background
    fi
}
stop_probe() {
    printf 'stopped\n' > /run/lpic/maintenance-probe-stopped
    event stopped
    kill -STOP "$$"
}
trap stop_probe TSTP
trap record_resume CONT
trap 'printf "terminated\n" > /run/lpic/maintenance-probe-terminated; event terminated; exit 0' TERM
for i in $(seq 1 250); do
    record_foreground && break
    sleep 0.02
done
printf 'ready\n' > /run/lpic/maintenance-probe-ready
while :; do
    sleep 2 &
    wait "$!"
done
EOF
chmod 0755 /usr/local/bin/lpic-maintenance-job-probe
cat > /usr/local/bin/lpic-reload-worker <<'EOF'
#!/usr/bin/env bash
set -u
trap 'printf "reload %s\n" "$$" > /run/lpic/reload-requested' USR1
trap 'exit 0' TERM
while :; do
    sleep 2 &
    wait "$!"
done
EOF
chmod 0755 /usr/local/bin/lpic-reload-worker
nohup bash -c 'exec -a reload-worker /usr/bin/bash /usr/local/bin/lpic-reload-worker' >/dev/null 2>&1 &
printf '%s\n' "$!" > /run/lpic/reload-worker.pid
sleep 0.1
