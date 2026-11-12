#!/usr/bin/env bash
set -Eeuo pipefail
trap 'code=$?; printf "reference failure at line %s (exit %s): %s\n" "$LINENO" "$code" "$BASH_COMMAND" >&2' ERR
reload_pid="$(pgrep -f '^reload-worker' | head -n1)"
test -n "$reload_pid"
ps -o pid=,ppid=,stat=,args= -p "$reload_pid" > /run/lpic/maintenance-inspection
kill -USR1 "$reload_pid"
for i in $(seq 1 100); do test -s /run/lpic/reload-requested && break; sleep 0.02; done
test "$(cat /run/lpic/reload-requested)" = "reload $reload_pid"
cat > /run/lpic/.lpic-reference-job-control.sh <<'LPIC_JOB'
#!/usr/bin/env bash
set -Eeuo pipefail
set -m
/usr/local/bin/lpic-maintenance-job-probe &
probe_pid="$!"
(
  for i in $(seq 1 250); do
    test -s /run/lpic/maintenance-probe-ready && break
    sleep 0.02
  done
  test -s /run/lpic/maintenance-probe-ready
  kill -TSTP "$probe_pid"
) &
fg %1 || true
test -f /run/lpic/maintenance-probe-foreground
test -f /run/lpic/maintenance-probe-stopped
jobs -s | grep -Fq lpic-maintenance-job-probe
bg %1
jobs -r | grep -Fq lpic-maintenance-job-probe
for i in $(seq 1 250); do test -f /run/lpic/maintenance-probe-background && break; sleep 0.02; done
test -f /run/lpic/maintenance-probe-background
kill -TERM "$probe_pid"
wait "$probe_pid" || true
LPIC_JOB
script -q -e -c '/usr/bin/bash -i /run/lpic/.lpic-reference-job-control.sh' /dev/null
rm -f /run/lpic/.lpic-reference-job-control.sh
