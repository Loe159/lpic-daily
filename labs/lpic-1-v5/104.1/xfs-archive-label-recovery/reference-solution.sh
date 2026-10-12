#!/usr/bin/env bash
set -euo pipefail
xfs_repair -n /dev/vdb >/dev/null
xfs_admin -L archive2026 /dev/vdb >/dev/null
mount /srv/xfsarchive
