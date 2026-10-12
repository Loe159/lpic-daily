#!/usr/bin/env bash
set -euo pipefail
command -v udevadm >/dev/null
test -d /sys/class/block && test -b /dev/vdb
mkfs.ext4 -F -q -L LPICBACKUP /dev/vdb
printf 'backup-manifest=retain-62\n' > /tmp/lpic-manifest.txt
debugfs -w -R 'write /tmp/lpic-manifest.txt /manifest.txt' /dev/vdb >/dev/null 2>&1
cat > /usr/local/bin/lpic-backup-read <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
test -b /dev/lpic-backup
test "$(blkid -s LABEL -o value /dev/lpic-backup)" = LPICBACKUP
printf 'backup-volume=ready\n'
EOF
chmod 0755 /usr/local/bin/lpic-backup-read
if test -e /dev/lpic-backup; then
  echo 'Expected persistent disk alias missing on initial state' >&2
  exit 1
fi
