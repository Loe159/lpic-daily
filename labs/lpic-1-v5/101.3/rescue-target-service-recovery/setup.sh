#!/usr/bin/env bash
set -euo pipefail
test -d /run/systemd/system
systemctl is-active --quiet qemu-guest-agent.service
command -v crond >/dev/null
systemctl cat rescue.target >/dev/null
systemctl cat multi-user.target >/dev/null

# Keep the remote lab control channel alive while isolating the guest.
install -d -m 0755 /etc/systemd/system/qemu-guest-agent.service.d
cat > /etc/systemd/system/qemu-guest-agent.service.d/95-lpic-isolation-guard.conf <<'EOF'
[Unit]
IgnoreOnIsolate=yes
EOF
systemctl daemon-reload
test "$(systemctl show qemu-guest-agent.service -p IgnoreOnIsolate --value)" = yes
systemctl is-active --quiet qemu-guest-agent.service

systemctl set-default multi-user.target
systemctl enable --now crond.service
systemctl is-active --quiet crond.service

systemctl --no-block isolate rescue.target
for attempt in $(seq 1 100); do
  if systemctl is-active --quiet rescue.target && ! systemctl is-active --quiet crond.service; then
    break
  fi
  sleep 0.2
done
systemctl is-active --quiet rescue.target
! systemctl is-active --quiet multi-user.target
! systemctl is-active --quiet crond.service
systemctl is-active --quiet qemu-guest-agent.service
