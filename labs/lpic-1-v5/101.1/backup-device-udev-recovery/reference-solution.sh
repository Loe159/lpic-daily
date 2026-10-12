#!/usr/bin/env bash
set -euo pipefail
cat > /etc/udev/rules.d/85-lpic-backup.rules <<'EOF'
SUBSYSTEM=="block", ENV{ID_FS_LABEL}=="LPICBACKUP", SYMLINK+="lpic-backup"
EOF
udevadm control --reload-rules
udevadm trigger --action=change --subsystem-match=block
udevadm settle
