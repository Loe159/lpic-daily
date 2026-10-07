#!/usr/bin/env bash
set -euo pipefail
/usr/bin/bash --noprofile --rcfile /root/.bashrc -ic '
tool=/opt/lpic/approved/bin/report-status
"$tool" "$REPORT_FILE"
printf "%s\nready\n" "$tool" > /run/lpic/external-command-summary
'
