#!/usr/bin/env bash
set -euo pipefail
sed -i 's/^write_file:/write_files:/' /var/lib/cloud/seed/nocloud/user-data
cloud-init clean --logs
cloud-init init --local
cloud-init init
cloud-init modules --mode=config
cloud-init modules --mode=final
test "$(cat /etc/lpic-cloud/agent.conf)" = agent=ready
