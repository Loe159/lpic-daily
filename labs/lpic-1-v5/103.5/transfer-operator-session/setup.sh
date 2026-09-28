#!/usr/bin/env bash
set -euo pipefail

install -d -m 0755 /run/lpic

cat > /usr/local/bin/lpic-maintenance-job-probe <<'EOF'
#!/usr/bin/env bash
set -u

record_resume() {
    printf 'continued\n' > /run/lpic/maintenance-probe-continued
    own_pgid="$(ps -o pgid= -p "$$" | tr -d ' ')"
    terminal_pgid="$(ps -o tpgid= -p "$$" | tr -d ' ')"
    if [[ -n "$own_pgid" && -n "$terminal_pgid" && "$own_pgid" != "$terminal_pgid" ]]; then
        printf 'background\n' > /run/lpic/maintenance-probe-background
    fi
}

trap 'printf "stopped\n" > /run/lpic/maintenance-probe-stopped; kill -STOP "$$"' TSTP
trap 'record_resume' CONT
trap 'printf "terminated\n" > /run/lpic/maintenance-probe-terminated; exit 0' TERM

while :; do
    sleep 2 &
    wait "$!"
done
EOF
chmod 0755 /usr/local/bin/lpic-maintenance-job-probe

cat > /usr/local/bin/lpic-reload-worker <<'EOF'
#!/usr/bin/env bash
set -u
trap 'printf "reload\n" > /run/lpic/reload-requested' USR1
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
