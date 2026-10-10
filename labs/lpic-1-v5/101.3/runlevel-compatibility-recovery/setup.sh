#!/usr/bin/env bash
set -euo pipefail
command -v runlevel >/dev/null
test -d /run/systemd/system
systemctl set-default graphical.target
systemctl isolate graphical.target
test "$(systemctl get-default)" = graphical.target
systemctl is-active --quiet graphical.target
systemctl is-active --quiet qemu-guest-agent.service
