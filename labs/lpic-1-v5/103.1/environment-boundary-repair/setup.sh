#!/usr/bin/env bash
set -euo pipefail
install -d -m 0755 /run/lpic
cat > /run/lpic/.lpic-env-capture.sh <<'EOF'
# Snapshot the environment of the learner's currently running shell.
__lpic_capture_environment() {
    local failed=0 child summary
    if [[ ${LOCAL_NOTE-} != operator-note ]]; then failed=$((failed | 1)); fi
    child=$'\n'$(/usr/bin/env)$'\n'
    if [[ "$child" == *$'\nLOCAL_NOTE='* ]]; then failed=$((failed | 2)); fi
    if [[ "$child" != *$'\nSERVICE_MODE=production\n'* ]]; then failed=$((failed | 4)); fi
    if [[ ${LEGACY_TOKEN+x} ]] || [[ "$child" == *$'\nLEGACY_TOKEN='* ]]; then failed=$((failed | 8)); fi
    summary=$(/usr/bin/cat /run/lpic/env-summary 2>/dev/null; /usr/bin/printf '\036')
    if [[ "$summary" != $'production|operator-note\036' &&
          "$summary" != $'production|operator-note\n\036' ]]; then
        failed=$((failed | 16))
    fi
    builtin printf '%d\n' "$failed" > /run/lpic/.environment-check-state
}
EOF
cat > /run/lpic/.lpic-shell-rc <<'EOF'
# Deliberately polluted initial shell environment; no startup file edits needed.
SERVICE_MODE=staging
export LOCAL_NOTE=operator-note
export LEGACY_TOKEN=deprecated
source /run/lpic/.lpic-env-capture.sh
PROMPT_COMMAND=__lpic_capture_environment
EOF
rm -f /run/lpic/env-summary /run/lpic/.environment-check-state
