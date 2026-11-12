#!/usr/bin/env bash
set -euo pipefail
install -d -m 0755 /run/lpic
cat > /usr/local/bin/lpic-batch-worker <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
pgid="$(ps -o pgid= -p "$$" | tr -d ' ')"
tpgid="$(ps -o tpgid= -p "$$" | tr -d ' ')"
if [[ "$tpgid" == "-1" || "$tpgid" == "0" || ( -n "$pgid" && "$pgid" != "$tpgid" ) ]]; then
    printf 'background\n' > /run/lpic/batch-worker-mode
fi
exec -a lpic-batch-worker /usr/bin/sleep infinity
EOF
chmod 0755 /usr/local/bin/lpic-batch-worker
cat > /usr/local/bin/lpic-handoff-daemon <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
printf 'handoff-daemon ready\n'
exec -a lpic-handoff-daemon /usr/bin/sleep infinity
EOF
chmod 0755 /usr/local/bin/lpic-handoff-daemon
