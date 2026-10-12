#!/usr/bin/env bash
set -euo pipefail
# Correct the current shell's environment; startup-file edits are unnecessary.
source /root/.bashrc
source /run/lpic/.lpic-env-capture.sh
export SERVICE_MODE=production
export -n LOCAL_NOTE
unset LEGACY_TOKEN
printf '%s|%s\n' "$SERVICE_MODE" "$LOCAL_NOTE" > /run/lpic/env-summary
__lpic_capture_environment
