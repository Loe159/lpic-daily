#!/usr/bin/env bash
set -euo pipefail
systemctl start lpic-maintenance.target
systemctl is-active --quiet lpic-maintenance.target
test "$(cat /var/lib/lpic-maintenance/stop-event)" = worker=stopped-gracefully
