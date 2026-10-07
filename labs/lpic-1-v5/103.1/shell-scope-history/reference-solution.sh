#!/usr/bin/env bash
set -euo pipefail
LOCAL_ONLY=inside
export CHILD_VISIBLE=exported
export REMOVE_ME=remove
set | grep '^LOCAL_ONLY=' > /run/lpic/set-local.txt
env | grep '^LOCAL_ONLY=' > /run/lpic/env-local.txt || true
env | grep '^CHILD_VISIBLE=' > /run/lpic/env-exported.txt
unset REMOVE_ME
bash -c 'env | grep "^CHILD_VISIBLE="; if env | grep -q "^REMOVE_ME="; then exit 1; fi' > /run/lpic/child-env.txt
/workspace/tools/outside-tool > /run/lpic/outside-result.txt
printf 'echo LPIC_HISTORY_MARKER\n' >> /root/.bash_history
REPORT_NAME=daily
echo "report:$REPORT_NAME" > /run/lpic/expansion.txt
