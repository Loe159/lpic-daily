#!/usr/bin/env bash
set -euo pipefail

install -d -m 0755 /run/lpic

cat > /usr/local/bin/lpic-signal-probe <<'EOF'
#!/usr/bin/env bash
set -u

trap 'printf "continued\n" > /run/lpic/probe-continued' CONT
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
