#!/usr/bin/env bash
set -euo pipefail
systemctl --no-block isolate multi-user.target
for attempt in $(seq 1 150); do
  if systemctl is-active --quiet multi-user.target &&
     ! systemctl is-active --quiet rescue.target &&
     systemctl is-active --quiet crond.service; then
    break
  fi
  sleep 0.2
done
systemctl is-active --quiet multi-user.target
! systemctl is-active --quiet rescue.target
systemctl is-active --quiet crond.service
systemctl is-active --quiet qemu-guest-agent.service
