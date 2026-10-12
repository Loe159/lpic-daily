#!/usr/bin/env bash
set -euo pipefail
command -v modprobe >/dev/null
command -v modinfo >/dev/null
test -s "$(modinfo -n brd)"
if lsmod | awk 'NR>1 && $1=="brd"{found=1} END{exit !found}'; then
  modprobe -r brd
fi
test ! -d /sys/module/brd
install -d -m 0755 /var/lib/lpic-ramcache /run/lpic-ramcache
printf 'archive-batch=preserve-81\n' > /var/lib/lpic-ramcache/source.txt
cat > /etc/systemd/system/lpic-ramcache-reader.service <<'EOF'
[Unit]
Description=LPIC volatile cache reader
After=local-fs.target
[Service]
Type=oneshot
ExecStart=/usr/bin/cmp /var/lib/lpic-ramcache/source.txt /run/lpic-ramcache/source.txt
RemainAfterExit=yes
[Install]
WantedBy=multi-user.target
EOF
systemctl daemon-reload
systemctl enable lpic-ramcache-reader.service >/dev/null
systemctl start lpic-ramcache-reader.service >/dev/null 2>&1 || true
test ! -b /dev/ram0
