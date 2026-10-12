#!/usr/bin/env bash
set -euo pipefail
test -b /dev/vdb
command -v nmcli >/dev/null
systemctl is-active --quiet NetworkManager.service
install -d -m 0755 /srv/lpic-virtual-journal /var/lib/lpic-vm-resources
iface=$(nmcli -t -f DEVICE,TYPE device status | awk -F: '$2=="ethernet" && $1!="lo"{print $1; exit}')
test -n "$iface"
ip -4 -o address show dev "$iface" scope global | grep -q 'inet '
printf '%s\n' "$iface" > /var/lib/lpic-vm-resources/iface
mkfs.ext4 -F -q -L LPICJOURNAL /dev/vdb
printf 'virtual-journal=retain-44\n' > /tmp/lpic-virtual-journal.txt
debugfs -w -R 'write /tmp/lpic-virtual-journal.txt /original.log' /dev/vdb >/dev/null 2>&1
printf 'LABEL=LPICJOURNAL /srv/lpic-virtual-journal ext4 defaults,nofail 0 2\n' >> /etc/fstab
cat > /usr/local/bin/lpic-virtual-journal-read <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
iface=$(cat /var/lib/lpic-vm-resources/iface)
mountpoint -q /srv/lpic-virtual-journal
grep -Fxq virtual-journal=retain-44 /srv/lpic-virtual-journal/original.log
ip -4 -o address show dev "$iface" scope global | grep -q 'inet '
EOF
chmod 0755 /usr/local/bin/lpic-virtual-journal-read
cat > /etc/systemd/system/lpic-virtual-journal.service <<'EOF'
[Unit]
Description=LPIC virtual resource journal reader
[Service]
Type=oneshot
ExecStart=/usr/local/bin/lpic-virtual-journal-read
RemainAfterExit=yes
EOF
systemctl daemon-reload
nmcli --wait 15 device disconnect "$iface"
systemctl start lpic-virtual-journal.service >/dev/null 2>&1 || true
test ! -e /srv/lpic-virtual-journal/original.log
