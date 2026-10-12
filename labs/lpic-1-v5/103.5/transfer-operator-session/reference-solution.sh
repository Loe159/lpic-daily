#!/usr/bin/env bash
set -euo pipefail

reload_pid="$(pgrep -f '^reload-worker' | head -n1)"
test -n "$reload_pid"
ps -o pid=,ppid=,stat=,comm=,args= -p "$reload_pid" > /run/lpic/maintenance-inspection
kill -USR1 "$reload_pid"
for _ in $(seq 1 50); do [[ -f /run/lpic/reload-requested ]] && break; sleep 0.02; done
test -f /run/lpic/reload-requested

# Exercise actual interactive job control in a disposable pseudo-terminal.
# A direct kill -CONT can bypass the 'jobs'/'bg' competency completely.
command -v script >/dev/null
cat > /run/lpic/.lpic-reference-job-control.sh <<'LPIC_JOB'
#!/usr/bin/env bash
set -euo pipefail
set -m
/usr/local/bin/lpic-maintenance-job-probe &
probe_pid="$!"
sleep 0.2
kill -TSTP "$probe_pid"
# Bash runs signal traps only once its child wait returns; allow > 2 seconds.
for i in $(seq 1 250); do test -f /run/lpic/maintenance-probe-stopped && break; sleep 0.02; done
test -f /run/lpic/maintenance-probe-stopped
jobs -s | grep -Fq "lpic-maintenance-job-probe"
bg %1
jobs -r | grep -Fq "lpic-maintenance-job-probe"
for i in $(seq 1 250); do test -f /run/lpic/maintenance-probe-background && break; sleep 0.02; done
test -f /run/lpic/maintenance-probe-background
kill -TERM "$probe_pid"
wait "$probe_pid" || true
LPIC_JOB
script -q -e -c '/usr/bin/bash -i /run/lpic/.lpic-reference-job-control.sh' /dev/null
rm -f /run/lpic/.lpic-reference-job-control.sh

bash -c 'exec -a batch-worker sleep infinity' &
nohup bash -c 'exec -a handoff-daemon sleep infinity' >/run/lpic/handoff-daemon.log 2>&1 &
tmux new-session -d -s maintenance-ops 'sleep infinity'
