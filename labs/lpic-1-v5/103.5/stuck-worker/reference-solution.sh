#!/usr/bin/env bash
set -Eeuo pipefail
trap 'code=$?; printf "reference failure at line %s (exit %s): %s\n" "$LINENO" "$code" "$BASH_COMMAND" >&2' ERR
ps -o pid=,ppid=,stat=,comm=,args= \
  -p "$(cat /run/lpic/healthy-worker.pid),$(cat /run/lpic/stuck-worker.pid)" \
  > /run/lpic/process-inspection
kill -TERM "$(cat /run/lpic/stuck-worker.pid)"
# Exercise actual interactive job control in a disposable pseudo-terminal.
# A direct kill -CONT can bypass the 'jobs'/'bg' competency completely.
command -v script >/dev/null
cat > /run/lpic/.lpic-reference-job-control.sh <<'LPIC_JOB'
#!/usr/bin/env bash
set -Eeuo pipefail
trap 'code=$?; printf "job-control failure at line %s (exit %s): %s\n" "$LINENO" "$code" "$BASH_COMMAND" >&2' ERR
set -m
/usr/local/bin/lpic-signal-probe &
probe_pid="$!"
for i in $(seq 1 250); do
  test -f /run/lpic/probe-ready && break
  kill -0 "$probe_pid" || { echo "signal probe exited before readiness" >&2; exit 1; }
  sleep 0.02
done
test -f /run/lpic/probe-ready || { echo "signal probe never became ready" >&2; exit 1; }
kill -TSTP "$probe_pid"
# Bash runs signal traps only once its child wait returns; allow > 2 seconds.
for i in $(seq 1 250); do test -f /run/lpic/probe-stopped && break; sleep 0.02; done
test -f /run/lpic/probe-stopped
jobs -s | grep -Fq "lpic-signal-probe"
bg %1
jobs -r | grep -Fq "lpic-signal-probe"
for i in $(seq 1 250); do test -f /run/lpic/probe-background && break; sleep 0.02; done
test -f /run/lpic/probe-background
kill -TERM "$probe_pid"
wait "$probe_pid" || true
LPIC_JOB
script -q -e -c '/usr/bin/bash -i /run/lpic/.lpic-reference-job-control.sh' /dev/null
rm -f /run/lpic/.lpic-reference-job-control.sh

nohup bash -c 'exec -a resilient-worker sleep infinity' >/run/lpic/resilient.log 2>&1 &
tmux new-session -d -s ops 'sleep infinity'
